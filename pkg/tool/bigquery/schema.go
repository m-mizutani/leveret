package bigquery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/genai"
)

// executeSchema executes bigquery_schema
func (t *Tool) executeSchema(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	type input struct {
		Project   string `json:"project"`
		DatasetID string `json:"dataset_id"`
		Table     string `json:"table"`
	}

	paramsJSON, err := json.Marshal(fc.Args)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal function arguments")
	}

	var in input
	if err := json.Unmarshal(paramsJSON, &in); err != nil {
		return nil, goerr.Wrap(err, "failed to parse input parameters")
	}

	if in.Project == "" {
		return nil, goerr.New("project is required")
	}
	if in.DatasetID == "" {
		return nil, goerr.New("dataset_id is required")
	}
	if in.Table == "" {
		return nil, goerr.New("table is required")
	}

	metadata, err := t.bq.GetTableMetadata(ctx, in.Project, in.DatasetID, in.Table)
	if err != nil {
		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"error": fmt.Sprintf("Failed to get table metadata: %v", err),
			},
		}, nil
	}

	formattedSchema := formatSchema(metadata.Schema, 0)

	response := map[string]any{
		"project":     in.Project,
		"dataset_id":  in.DatasetID,
		"table":       in.Table,
		"full_table":  fmt.Sprintf("%s.%s.%s", in.Project, in.DatasetID, in.Table),
		"schema":      formattedSchema,
		"field_count": len(metadata.Schema),
		"num_rows":    metadata.NumRows,
		"num_bytes":   metadata.NumBytes,
	}

	// Add partition information if available
	if metadata.TimePartitioning != nil {
		partitionInfo := map[string]any{
			"type": metadata.TimePartitioning.Type,
		}
		if metadata.TimePartitioning.Field != "" {
			partitionInfo["field"] = metadata.TimePartitioning.Field
		}
		if metadata.TimePartitioning.Expiration != 0 {
			partitionInfo["expiration_ms"] = metadata.TimePartitioning.Expiration.Milliseconds()
		}
		response["time_partitioning"] = partitionInfo
	}

	// Add range partition information if available
	if metadata.RangePartitioning != nil {
		rangeInfo := map[string]any{
			"field": metadata.RangePartitioning.Field,
		}
		if metadata.RangePartitioning.Range != nil {
			rangeInfo["range"] = map[string]any{
				"start":    metadata.RangePartitioning.Range.Start,
				"end":      metadata.RangePartitioning.Range.End,
				"interval": metadata.RangePartitioning.Range.Interval,
			}
		}
		response["range_partitioning"] = rangeInfo
	}

	// Add clustering information if available
	if metadata.Clustering != nil && len(metadata.Clustering.Fields) > 0 {
		response["clustering_fields"] = metadata.Clustering.Fields
	}

	return &genai.FunctionResponse{
		Name:     fc.Name,
		Response: response,
	}, nil
}

// formatSchema formats the schema fields as a readable string
func formatSchema(fields []*bigquery.FieldSchema, indent int) string {
	var lines []string
	prefix := strings.Repeat("  ", indent)

	for _, field := range fields {
		line := fmt.Sprintf("%s- %s (%s)", prefix, field.Name, field.Type)

		if field.Required {
			line += " [REQUIRED]"
		}
		if field.Repeated {
			line += " [REPEATED]"
		}
		if field.Description != "" {
			line += fmt.Sprintf(" - %s", field.Description)
		}

		lines = append(lines, line)

		if len(field.Schema) > 0 {
			lines = append(lines, formatSchema(field.Schema, indent+1))
		}
	}

	return strings.Join(lines, "\n")
}
