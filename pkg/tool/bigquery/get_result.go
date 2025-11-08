package bigquery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/genai"
)

// executeGetResult executes bigquery_get_result
func (t *Tool) executeGetResult(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	type input struct {
		JobID  string `json:"job_id"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}

	paramsJSON, err := json.Marshal(fc.Args)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal function arguments")
	}

	var in input
	if err := json.Unmarshal(paramsJSON, &in); err != nil {
		return nil, goerr.Wrap(err, "failed to parse input parameters")
	}

	if in.JobID == "" {
		return nil, goerr.New("job_id is required")
	}

	if in.Limit <= 0 {
		in.Limit = 100
	}
	if in.Limit > int(t.resultLimitRows) {
		in.Limit = int(t.resultLimitRows)
	}
	if in.Offset < 0 {
		in.Offset = 0
	}

	// Load results from memory
	allResults, exists := t.results[in.JobID]
	if !exists {
		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"error": fmt.Sprintf("Job ID '%s' not found. The job may not exist or results may have been cleared.", in.JobID),
			},
		}, nil
	}

	// Apply pagination
	totalRows := len(allResults)
	start := in.Offset
	end := in.Offset + in.Limit

	if start >= totalRows {
		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"total_rows": totalRows,
				"returned":   0,
				"offset":     in.Offset,
				"limit":      in.Limit,
				"rows":       []map[string]any{},
				"message":    fmt.Sprintf("Offset %d is beyond total rows %d", in.Offset, totalRows),
			},
		}, nil
	}

	if end > totalRows {
		end = totalRows
	}

	paginatedResults := allResults[start:end]
	resultsJSON, err := json.MarshalIndent(paginatedResults, "", "  ")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to format results")
	}

	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"total_rows": totalRows,
			"returned":   len(paginatedResults),
			"offset":     in.Offset,
			"limit":      in.Limit,
			"rows":       string(resultsJSON),
			"has_more":   end < totalRows,
		},
	}, nil
}
