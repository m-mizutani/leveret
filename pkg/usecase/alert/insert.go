package alert

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"text/template"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/utils/logging"
	"google.golang.org/genai"
)

func (u *UseCase) Insert(
	ctx context.Context,
	data any,
) (*model.Alert, error) {
	alert := &model.Alert{
		ID:        model.NewAlertID(),
		Data:      data,
		CreatedAt: time.Now(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal alert data")
	}

	title, err := generateTitle(ctx, u.gemini, string(jsonData), 100)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate title")
	}
	alert.Title = title

	description, err := generateDescription(ctx, u.gemini, string(jsonData))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate description")
	}
	alert.Description = description

	if err := u.repo.PutAlert(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

//go:embed prompt/title.md
var titlePromptRaw string

var titlePromptTmpl = template.Must(template.New("title").Parse(titlePromptRaw))

func generateTitle(ctx context.Context, gemini adapter.Gemini, alertData string, maxLength int) (string, error) {
	const maxRetries = 3
	var failedExamples []struct {
		Title  string
		Length int
	}
	logger := logging.From(ctx)

	for attempt := 0; attempt < maxRetries; attempt++ {
		var buf bytes.Buffer
		if err := titlePromptTmpl.Execute(&buf, map[string]any{
			"MaxLength":      maxLength,
			"AlertData":      alertData,
			"FailedExamples": failedExamples,
		}); err != nil {
			return "", goerr.Wrap(err, "failed to execute title prompt template")
		}

		contents := []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{{Text: buf.String()}},
			},
		}

		resp, err := gemini.GenerateContent(ctx, contents, nil)
		if err != nil {
			return "", goerr.Wrap(err, "failed to generate content for title")
		}

		if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
			return "", goerr.New("invalid response structure from gemini")
		}

		title := strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text)
		if len(title) <= maxLength {
			logger.Debug("title accepted", "title", title)
			return title, nil
		}

		logger.Warn("title too long, retrying", "title", title, "length", len(title), "maxLength", maxLength)
		failedExamples = append(failedExamples, struct {
			Title  string
			Length int
		}{title, len(title)})
	}

	return "", goerr.New("failed to generate title within character limit after retries")
}

func generateDescription(ctx context.Context, gemini adapter.Gemini, alertData string) (string, error) {
	prompt := "Generate a detailed description (2-3 sentences) for this security alert. Explain what happened and why it might be important. Return only the description without any explanation:\n\n" + alertData

	contents := []*genai.Content{
		{
			Role:  "user",
			Parts: []*genai.Part{{Text: prompt}},
		},
	}

	resp, err := gemini.GenerateContent(ctx, contents, nil)
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate content for description")
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", goerr.New("invalid response structure from gemini")
	}

	return strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text), nil
}
