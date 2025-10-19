package alert

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
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

	summary, err := generateSummary(ctx, u.gemini, string(jsonData))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate summary")
	}
	alert.Title = summary.Title
	alert.Description = summary.Description

	if err := u.repo.PutAlert(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

//go:embed prompt/summary.md
var summaryPromptRaw string

var summaryPromptTmpl = template.Must(template.New("summary").Parse(summaryPromptRaw))

const maxTitleLength = 100

type alertSummary struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (s *alertSummary) validate() error {
	if len(s.Title) > maxTitleLength {
		return goerr.New("title too long", goerr.V("title", s.Title), goerr.V("length", len(s.Title)), goerr.V("maxLength", maxTitleLength))
	}
	if s.Title == "" {
		return goerr.New("title is empty")
	}
	if s.Description == "" {
		return goerr.New("description is empty")
	}
	return nil
}

func generateSummary(ctx context.Context, gemini adapter.Gemini, alertData string) (*alertSummary, error) {
	const maxRetries = 3
	var failedExamples []string
	logger := logging.From(ctx)

	for attempt := 0; attempt < maxRetries; attempt++ {
		var buf bytes.Buffer
		if err := summaryPromptTmpl.Execute(&buf, map[string]any{
			"MaxTitleLength": maxTitleLength,
			"AlertData":      alertData,
			"FailedExamples": failedExamples,
		}); err != nil {
			return nil, goerr.Wrap(err, "failed to execute summary prompt template")
		}

		contents := []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{{Text: buf.String()}},
			},
		}

		maxLen := int64(maxTitleLength)
		config := &genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"title": {
						Type:        genai.TypeString,
						Description: "Short title for the alert",
						MaxLength:   &maxLen,
					},
					"description": {
						Type:        genai.TypeString,
						Description: "Detailed description (2-3 sentences) for the alert",
					},
				},
				Required: []string{"title", "description"},
			},
		}

		resp, err := gemini.GenerateContent(ctx, contents, config)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to generate content for summary")
		}

		if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
			return nil, goerr.New("invalid response structure from gemini", goerr.V("resp", resp))
		}

		// DEBUG: remove me later
		rawJSON := resp.Candidates[0].Content.Parts[0].Text
		fmt.Printf("[DEBUG] received JSON from Gemini: %s", rawJSON)

		var summary alertSummary
		if err := json.Unmarshal([]byte(rawJSON), &summary); err != nil {
			return nil, goerr.Wrap(err, "failed to unmarshal summary JSON", goerr.V("text", rawJSON))
		}

		if err := summary.validate(); err != nil {
			logger.Warn("validation failed, retrying", "error", err, "title", summary.Title)
			failedExamples = append(failedExamples, err.Error())
			continue
		}

		logger.Debug("summary accepted", "title", summary.Title, "description", summary.Description)
		return &summary, nil
	}

	return nil, goerr.New("failed to generate valid summary after retries")
}
