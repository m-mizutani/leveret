package chat

import (
	"bytes"
	"context"
	_ "embed"
	"text/template"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/genai"
)

//go:embed prompt/conclude.md
var concludePromptRaw string

var concludePromptTmpl = template.Must(template.New("conclude").Funcs(template.FuncMap{
	"add": func(a, b int) int { return a + b },
}).Parse(concludePromptRaw))

// Generate creates a comprehensive conclusion from all investigation results
func (c *conclusionGenerator) Generate(ctx context.Context, plan *Plan, results []*StepResult, reflections []*Reflection) (*Conclusion, error) {
	// Build prompt using template
	var buf bytes.Buffer
	if err := concludePromptTmpl.Execute(&buf, map[string]any{
		"Objective":   plan.Objective,
		"Steps":       plan.Steps,
		"Results":     results,
		"Reflections": reflections,
	}); err != nil {
		return nil, goerr.Wrap(err, "failed to execute conclude prompt template")
	}

	// Build config (no structured output, just plain text)
	thinkingBudget := int32(0)
	config := &genai.GenerateContentConfig{
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: false,
			ThinkingBudget:  &thinkingBudget,
		},
	}

	// Create content
	contents := []*genai.Content{
		genai.NewContentFromText(buf.String(), genai.RoleUser),
	}

	// Generate conclusion
	resp, err := c.gemini.GenerateContent(ctx, contents, config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate conclusion")
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, goerr.New("invalid response structure from gemini")
	}

	// Extract text directly from response
	var contentParts []string
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			contentParts = append(contentParts, part.Text)
		}
	}

	content := ""
	if len(contentParts) > 0 {
		content = contentParts[0] // Use first text part
	}

	// Convert to Conclusion structure
	conclusion := &Conclusion{
		Content:     content,
		GeneratedAt: time.Now(),
	}

	return conclusion, nil
}
