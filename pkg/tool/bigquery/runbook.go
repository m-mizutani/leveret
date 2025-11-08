package bigquery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/genai"
)

// executeRunbook executes bigquery_runbook
func (t *Tool) executeRunbook(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	type input struct {
		RunBookID string `json:"runbook_id"`
	}

	paramsJSON, err := json.Marshal(fc.Args)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal function arguments")
	}

	var in input
	if err := json.Unmarshal(paramsJSON, &in); err != nil {
		return nil, goerr.Wrap(err, "failed to parse input parameters")
	}

	if in.RunBookID == "" {
		return nil, goerr.New("runbook_id is required")
	}

	// Get specific runBook
	rb, exists := t.runBooks[in.RunBookID]
	if !exists {
		availableIDs := make([]string, 0, len(t.runBooks))
		for id := range t.runBooks {
			availableIDs = append(availableIDs, id)
		}

		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"error":         fmt.Sprintf("Runbook '%s' not found", in.RunBookID),
				"available_ids": availableIDs,
			},
		}, nil
	}

	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"runbook_id":  rb.ID,
			"title":       rb.Title,
			"description": rb.Description,
			"sql":         rb.SQL,
		},
	}, nil
}
