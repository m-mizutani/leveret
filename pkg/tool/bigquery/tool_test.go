package bigquery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/m-mizutani/leveret/pkg/tool/bigquery"
	"google.golang.org/genai"
)

func TestBigQueryTool(t *testing.T) {
	projectID := os.Getenv("TEST_BIGQUERY_PROJECT")
	if projectID == "" {
		t.Skip("TEST_BIGQUERY_PROJECT is not set")
	}

	datasetID := os.Getenv("TEST_BIGQUERY_DATASET")
	if datasetID == "" {
		t.Skip("TEST_BIGQUERY_DATASET is not set")
	}

	table := os.Getenv("TEST_BIGQUERY_TABLE")
	if table == "" {
		t.Skip("TEST_BIGQUERY_TABLE is not set")
	}

	query := os.Getenv("TEST_BIGQUERY_QUERY")
	if query == "" {
		t.Skip("TEST_BIGQUERY_QUERY is not set")
	}

	ctx := context.Background()
	bqTool := bigquery.New()

	// Set up flags
	flags := bqTool.Flags()
	for _, flag := range flags {
		switch f := flag.(type) {
		case interface{ GetDestination() *string }:
			switch flag.Names()[0] {
			case "bigquery-project":
				*f.GetDestination() = projectID
			}
		case interface{ GetDestination() *int64 }:
			switch flag.Names()[0] {
			case "bigquery-scan-limit-mb":
				*f.GetDestination() = 1024
			case "bigquery-result-limit-rows":
				*f.GetDestination() = 1000
			}
		}
	}

	// Initialize tool
	enabled, err := bqTool.Init(ctx, &tool.Client{})
	gt.NoError(t, err)
	gt.True(t, enabled)

	t.Run("Query and GetResult", func(t *testing.T) {
		// Execute query
		queryResp, err := bqTool.Execute(ctx, genai.FunctionCall{
			Name: "bigquery_query",
			Args: map[string]any{
				"query": query,
			},
		})
		gt.NoError(t, err)
		gt.NotNil(t, queryResp)

		response := queryResp.Response
		if errMsg, hasError := response["error"]; hasError {
			t.Fatalf("Query failed: %v", errMsg)
		}

		jobID, ok := response["job_id"].(string)
		gt.True(t, ok)
		gt.NotEqual(t, "", jobID)
		t.Logf("Job ID: %s", jobID)

		// Get results
		getResultResp, err := bqTool.Execute(ctx, genai.FunctionCall{
			Name: "bigquery_get_result",
			Args: map[string]any{
				"job_id": jobID,
				"limit":  10,
				"offset": 0,
			},
		})
		gt.NoError(t, err)
		gt.NotNil(t, getResultResp)

		resultResponse := getResultResp.Response
		if errMsg, hasError := resultResponse["error"]; hasError {
			t.Fatalf("Get result failed: %v", errMsg)
		}

		results, ok := resultResponse["results"].([]map[string]any)
		gt.True(t, ok)
		t.Logf("Result count: %d", len(results))
	})

	t.Run("Schema", func(t *testing.T) {
		schemaResp, err := bqTool.Execute(ctx, genai.FunctionCall{
			Name: "bigquery_schema",
			Args: map[string]any{
				"project":    projectID,
				"dataset_id": datasetID,
				"table":      table,
			},
		})
		gt.NoError(t, err)
		gt.NotNil(t, schemaResp)

		response := schemaResp.Response
		if errMsg, hasError := response["error"]; hasError {
			t.Fatalf("Schema query failed: %v", errMsg)
		}

		schema, ok := response["schema"].(string)
		gt.True(t, ok)
		gt.NotEqual(t, "", schema)
		t.Logf("Schema:\n%s", schema)

		if timePartitioning, exists := response["time_partitioning"]; exists {
			t.Logf("Time partitioning: %v", timePartitioning)
		}

		if rangePartitioning, exists := response["range_partitioning"]; exists {
			t.Logf("Range partitioning: %v", rangePartitioning)
		}

		if clusteringFields, exists := response["clustering_fields"]; exists {
			t.Logf("Clustering fields: %v", clusteringFields)
		}
	})
}

func TestBigQueryToolWithRunbook(t *testing.T) {
	projectID := os.Getenv("TEST_BIGQUERY_PROJECT")
	if projectID == "" {
		t.Skip("TEST_BIGQUERY_PROJECT is not set")
	}

	// Create temporary runbook directory
	tmpDir := t.TempDir()
	runbookFile := filepath.Join(tmpDir, "test_runbook.sql")

	// Write test runbook
	runbookContent := `-- title: Test Query
-- description: A test query for BigQuery

SELECT 1 as test_value`

	err := os.WriteFile(runbookFile, []byte(runbookContent), 0644)
	gt.NoError(t, err)

	ctx := context.Background()
	bqTool := bigquery.New()

	// Set up flags
	flags := bqTool.Flags()
	for _, flag := range flags {
		switch f := flag.(type) {
		case interface{ GetDestination() *string }:
			switch flag.Names()[0] {
			case "bigquery-project":
				*f.GetDestination() = projectID
			case "bigquery-runbook-dir":
				*f.GetDestination() = tmpDir
			}
		case interface{ GetDestination() *int64 }:
			switch flag.Names()[0] {
			case "bigquery-scan-limit-mb":
				*f.GetDestination() = 1024
			case "bigquery-result-limit-rows":
				*f.GetDestination() = 1000
			}
		}
	}

	// Initialize tool
	enabled, err := bqTool.Init(ctx, &tool.Client{})
	gt.NoError(t, err)
	gt.True(t, enabled)

	// Check prompt contains runbook information
	prompt := bqTool.Prompt(ctx)
	gt.S(t, prompt).Contains("test_runbook")
	gt.S(t, prompt).Contains("Test Query")
	t.Logf("Prompt:\n%s", prompt)

	t.Run("Runbook", func(t *testing.T) {
		runbookResp, err := bqTool.Execute(ctx, genai.FunctionCall{
			Name: "bigquery_runbook",
			Args: map[string]any{
				"runbook_id": "test_runbook",
			},
		})
		gt.NoError(t, err)
		gt.NotNil(t, runbookResp)

		response := runbookResp.Response
		if errMsg, hasError := response["error"]; hasError {
			t.Fatalf("Runbook query failed: %v", errMsg)
		}

		sql, ok := response["sql"].(string)
		gt.True(t, ok)
		gt.S(t, sql).Contains("SELECT 1 as test_value")

		title, ok := response["title"].(string)
		gt.True(t, ok)
		gt.Equal(t, title, "Test Query")

		description, ok := response["description"].(string)
		gt.True(t, ok)
		gt.Equal(t, description, "A test query for BigQuery")
	})
}

func TestBigQueryToolWithTableList(t *testing.T) {
	projectID := os.Getenv("TEST_BIGQUERY_PROJECT")
	if projectID == "" {
		t.Skip("TEST_BIGQUERY_PROJECT is not set")
	}

	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Write test table list
	tableListContent := `tables:
  - project: project1
    dataset: dataset1
    table: table1
    description: Test table 1
  - project: project2
    dataset: dataset2
    table: table2
    description: Test table 2
  - project: project3
    dataset: dataset3
    table: table3
    description: Test table 3`

	err := os.WriteFile(configFile, []byte(tableListContent), 0644)
	gt.NoError(t, err)

	ctx := context.Background()
	bqTool := bigquery.New()

	// Set up flags
	flags := bqTool.Flags()
	for _, flag := range flags {
		switch f := flag.(type) {
		case interface{ GetDestination() *string }:
			switch flag.Names()[0] {
			case "bigquery-project":
				*f.GetDestination() = projectID
			case "bigquery-config-file":
				*f.GetDestination() = configFile
			}
		case interface{ GetDestination() *int64 }:
			switch flag.Names()[0] {
			case "bigquery-scan-limit-mb":
				*f.GetDestination() = 1024
			case "bigquery-result-limit-rows":
				*f.GetDestination() = 1000
			}
		}
	}

	// Initialize tool
	enabled, err := bqTool.Init(ctx, &tool.Client{})
	gt.NoError(t, err)
	gt.True(t, enabled)

	// Check prompt contains table list
	prompt := bqTool.Prompt(ctx)
	gt.S(t, prompt).Contains("project1.dataset1.table1")
	gt.S(t, prompt).Contains("project2.dataset2.table2")
	gt.S(t, prompt).Contains("project3.dataset3.table3")
	t.Logf("Prompt:\n%s", prompt)
}
