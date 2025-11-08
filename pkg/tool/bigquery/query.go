package bigquery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/genai"
)

// executeQuery executes bigquery_query
func (t *Tool) executeQuery(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	type input struct {
		Query string `json:"query"`
	}

	paramsJSON, err := json.Marshal(fc.Args)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal function arguments")
	}

	var in input
	if err := json.Unmarshal(paramsJSON, &in); err != nil {
		return nil, goerr.Wrap(err, "failed to parse input parameters")
	}

	if in.Query == "" {
		return nil, goerr.New("query is required")
	}

	// Perform dry run to check scan size
	bytesProcessed, err := t.bq.DryRun(ctx, in.Query)
	if err != nil {
		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"error": fmt.Sprintf("Query validation failed: %v", err),
			},
		}, nil
	}

	// Check scan limit
	scanLimitBytes := t.scanLimitMB * 1024 * 1024
	bytesProcessedMB := float64(bytesProcessed) / 1024 / 1024

	if bytesProcessed > scanLimitBytes {
		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"error": fmt.Sprintf(
					"Query would scan %.2f MB, which exceeds the limit of %d MB. Please refine your query to reduce data scanned (e.g., add date filters, limit columns, or use partitioned tables).",
					bytesProcessedMB,
					t.scanLimitMB,
				),
			},
		}, nil
	}

	// Execute query
	jobID, err := t.bq.Query(ctx, in.Query)
	if err != nil {
		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"error": fmt.Sprintf("Query execution failed: %v", err),
			},
		}, nil
	}

	// Get query results
	results, err := t.bq.GetQueryResult(ctx, jobID)
	if err != nil {
		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"error": fmt.Sprintf("Failed to retrieve query results: %v", err),
			},
		}, nil
	}

	// Store results in memory
	t.results[jobID] = results

	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"job_id":        jobID,
			"rows_returned": len(results),
		},
	}, nil
}
