package chat

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
	"github.com/m-mizutani/leveret/pkg/tool"
	"google.golang.org/genai"
)

//go:embed prompt/reflect.md
var reflectPromptRaw string

var reflectPromptTmpl = template.Must(template.New("reflect").Funcs(template.FuncMap{
	"add": func(a, b int) int { return a + b },
}).Parse(reflectPromptRaw))

// reflect evaluates a step's execution and determines if plan updates are needed
func reflect(ctx context.Context, gemini adapter.Gemini, registry *tool.Registry, step *Step, result *StepResult, plan *Plan) (*Reflection, error) {
	// Get available tools
	var toolDescriptions []string
	if registry != nil {
		for _, name := range registry.EnabledTools() {
			toolDescriptions = append(toolDescriptions, "- "+name)
		}
	}

	// Get existing plan steps summary
	completedSteps := []string{}
	pendingSteps := []string{}
	for _, s := range plan.Steps {
		if s.Status == StepStatusCompleted {
			completedSteps = append(completedSteps, s.Description)
		} else if s.Status == StepStatusPending {
			pendingSteps = append(pendingSteps, s.Description)
		}
	}

	// Build prompt using template
	var buf bytes.Buffer
	if err := reflectPromptTmpl.Execute(&buf, map[string]any{
		"StepID":          step.ID,
		"StepDescription": step.Description,
		"StepExpected":    step.Expected,
		"Success":         result.Success,
		"Findings":        result.Findings,
		"ToolCalls":       result.ToolCalls,
		"AvailableTools":  toolDescriptions,
		"CompletedSteps":  completedSteps,
		"PendingSteps":    pendingSteps,
	}); err != nil {
		return nil, goerr.Wrap(err, "failed to execute reflect prompt template")
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
				"achieved": {
					Type:        genai.TypeBoolean,
					Description: "Whether the step objective was achieved",
				},
				"insights": {
					Type:        genai.TypeArray,
					Description: "New insights or discoveries",
					Items: &genai.Schema{
						Type: genai.TypeString,
					},
				},
				"plan_updates": {
					Type:        genai.TypeArray,
					Description: "List of plan updates",
					Items: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"type": {
								Type:        genai.TypeString,
								Description: "Update type: add_step (add new step), update_step (update existing step), or cancel_step (cancel step)",
								Enum:        []string{"add_step", "update_step", "cancel_step"},
							},
							"step": {
								Type:        genai.TypeObject,
								Description: "Step information (for add_step/update_step)",
								Properties: map[string]*genai.Schema{
									"id": {
										Type:        genai.TypeString,
										Description: "Step ID",
									},
									"description": {
										Type:        genai.TypeString,
										Description: "Step description",
									},
									"tools": {
										Type:        genai.TypeArray,
										Description: "Expected tools",
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
							"step_id": {
								Type:        genai.TypeString,
								Description: "Step ID to cancel (for cancel_step)",
							},
							"reason": {
								Type:        genai.TypeString,
								Description: "Reason for cancellation (for cancel_step)",
							},
						},
						Required: []string{"type"},
					},
				},
			},
			Required: []string{"achieved", "insights", "plan_updates"},
		},
	}

	// Create content
	contents := []*genai.Content{
		genai.NewContentFromText(buf.String(), genai.RoleUser),
	}

	// Generate reflection
	resp, err := gemini.GenerateContent(ctx, contents, config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate reflection")
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, goerr.New("invalid response structure from gemini")
	}

	rawJSON := resp.Candidates[0].Content.Parts[0].Text

	// Parse JSON response
	var reflectionData struct {
		Achieved    bool     `json:"achieved"`
		Insights    []string `json:"insights"`
		PlanUpdates []struct {
			Type   string `json:"type"`
			Step   *struct {
				ID          string   `json:"id"`
				Description string   `json:"description"`
				Tools       []string `json:"tools"`
				Expected    string   `json:"expected"`
			} `json:"step,omitempty"`
			StepID string `json:"step_id,omitempty"`
			Reason string `json:"reason,omitempty"`
		} `json:"plan_updates"`
	}

	if err := json.Unmarshal([]byte(rawJSON), &reflectionData); err != nil {
		// Display error for debugging
		fmt.Printf("❌ Reflection JSON パースエラー: %v\n", err)
		fmt.Printf("   受信したJSON (全文):\n%s\n", rawJSON)
		return nil, goerr.Wrap(err, "failed to unmarshal reflection JSON", goerr.V("json", rawJSON))
	}

	// Convert to Reflection structure
	reflection := &Reflection{
		StepID:      step.ID,
		Achieved:    reflectionData.Achieved,
		Insights:    reflectionData.Insights,
		PlanUpdates: make([]PlanUpdate, 0, len(reflectionData.PlanUpdates)),
		ReflectedAt: time.Now(),
	}

	for _, updateData := range reflectionData.PlanUpdates {
		update := PlanUpdate{
			Type:   UpdateType(updateData.Type),
			StepID: updateData.StepID,
			Reason: updateData.Reason,
		}

		if updateData.Step != nil {
			update.Step = Step{
				ID:          updateData.Step.ID,
				Description: updateData.Step.Description,
				Tools:       updateData.Step.Tools,
				Expected:    updateData.Step.Expected,
				Status:      StepStatusPending,
			}
		}

		reflection.PlanUpdates = append(reflection.PlanUpdates, update)
	}

	return reflection, nil
}

// applyUpdates applies reflection updates to the plan
func applyUpdates(plan *Plan, reflection *Reflection) error {
	for _, update := range reflection.PlanUpdates {
		switch update.Type {
		case UpdateTypeAddStep:
			// Add step to the end
			plan.Steps = append(plan.Steps, update.Step)

		case UpdateTypeUpdateStep:
			// Find and replace step
			found := false
			for i, step := range plan.Steps {
				if step.ID == update.Step.ID {
					plan.Steps[i] = update.Step
					found = true
					break
				}
			}
			if !found {
				return goerr.New("target step not found for update_step", goerr.Value("step_id", update.Step.ID))
			}

		case UpdateTypeCancelStep:
			// Find and mark step as canceled
			found := false
			for i, step := range plan.Steps {
				if step.ID == update.StepID {
					plan.Steps[i].Status = StepStatusCanceled
					found = true
					break
				}
			}
			if !found {
				return goerr.New("target step not found for cancel_step", goerr.Value("step_id", update.StepID))
			}
		}
	}

	return nil
}
