package chat

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"text/template"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"google.golang.org/genai"
)

//go:embed prompt/plan.md
var planPromptRaw string

var planPromptTmpl = template.Must(template.New("plan").Parse(planPromptRaw))

// Generate creates an investigation plan from a user request
func (p *planGenerator) Generate(ctx context.Context, request string, alert *model.Alert, history []*genai.Content) (*Plan, error) {
	// Get available tools
	tools := p.registry.Tools()
	toolDescriptions := make([]string, 0)
	for _, t := range tools {
		spec := t.Spec()
		if spec == nil || spec.FunctionDeclarations == nil {
			continue
		}
		for _, fd := range spec.FunctionDeclarations {
			desc := fd.Description
			if desc == "" {
				desc = "(no description)"
			}
			toolDescriptions = append(toolDescriptions, "- **"+fd.Name+"**: "+desc)
		}
	}

	// Prepare alert data JSON
	alertDataJSON, err := json.MarshalIndent(alert.Data, "", "  ")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal alert data")
	}

	// Build prompt using template
	var buf bytes.Buffer
	if err := planPromptTmpl.Execute(&buf, map[string]any{
		"Request":          request,
		"AlertID":          alert.ID,
		"AlertTitle":       alert.Title,
		"AlertDescription": alert.Description,
		"AlertAttributes":  alert.Attributes,
		"AlertDataJSON":    string(alertDataJSON),
		"Tools":            toolDescriptions,
		"HasHistory":       len(history) > 0,
	}); err != nil {
		return nil, goerr.Wrap(err, "failed to execute plan prompt template")
	}

	// Build config with JSON schema for structured output
	thinkingBudget := int32(0)
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: false,
			ThinkingBudget:  &thinkingBudget,
		},
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"objective": {
					Type:        genai.TypeString,
					Description: "Investigation objective",
				},
				"steps": {
					Type:        genai.TypeArray,
					Description: "List of investigation steps",
					Items: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"id": {
								Type:        genai.TypeString,
								Description: "Step ID (e.g., step_1)",
							},
							"description": {
								Type:        genai.TypeString,
								Description: "Step description",
							},
							"tools": {
								Type:        genai.TypeArray,
								Description: "List of tools to use",
								Items: &genai.Schema{
									Type: genai.TypeString,
								},
							},
							"expected": {
								Type:        genai.TypeString,
								Description: "Expected outcome",
							},
						},
						Required: []string{"id", "description", "tools", "expected"},
					},
				},
			},
			Required: []string{"objective", "steps"},
		},
	}

	// Create content with history + new request
	contents := make([]*genai.Content, 0, len(history)+1)
	contents = append(contents, history...)
	contents = append(contents, genai.NewContentFromText(buf.String(), genai.RoleUser))

	// Generate plan
	resp, err := p.gemini.GenerateContent(ctx, contents, config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate plan")
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, goerr.New("invalid response structure from gemini")
	}

	rawJSON := resp.Candidates[0].Content.Parts[0].Text

	// Parse JSON response
	var planData struct {
		Objective string `json:"objective"`
		Steps     []struct {
			ID          string   `json:"id"`
			Description string   `json:"description"`
			Tools       []string `json:"tools"`
			Expected    string   `json:"expected"`
		} `json:"steps"`
	}

	if err := json.Unmarshal([]byte(rawJSON), &planData); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal plan JSON", goerr.Value("json", rawJSON))
	}

	// Convert to Plan structure
	plan := &Plan{
		Objective:   planData.Objective,
		Steps:       make([]Step, 0, len(planData.Steps)),
		GeneratedAt: time.Now(),
	}

	for _, stepData := range planData.Steps {
		step := Step{
			ID:          stepData.ID,
			Description: stepData.Description,
			Tools:       stepData.Tools,
			Expected:    stepData.Expected,
			Status:      StepStatusPending,
		}
		plan.Steps = append(plan.Steps, step)
	}

	return plan, nil
}
