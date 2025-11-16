package bigquery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/agent/bigquery"
	"github.com/m-mizutani/leveret/pkg/tool"
	"google.golang.org/genai"
)

func TestTool_Flags(t *testing.T) {
	bqTool := bigquery.New()
	flags := bqTool.Flags()

	// Should have 5 flags
	gt.V(t, len(flags)).Equal(5)

	flagNames := make(map[string]bool)
	for _, flag := range flags {
		flagNames[flag.Names()[0]] = true
	}

	gt.True(t, flagNames["bigquery-project"])
	gt.True(t, flagNames["bigquery-runbook-dir"])
	gt.True(t, flagNames["bigquery-config-file"])
	gt.True(t, flagNames["bigquery-scan-limit-mb"])
	gt.True(t, flagNames["bigquery-result-limit-rows"])
}

func TestTool_InitDisabledWithoutProject(t *testing.T) {
	ctx := context.Background()
	bqTool := bigquery.New()

	// Without setting project, tool should be disabled
	enabled, err := bqTool.Init(ctx, &tool.Client{})
	gt.NoError(t, err)
	gt.False(t, enabled)
}

func TestTool_Spec(t *testing.T) {
	bqTool := bigquery.New()
	spec := bqTool.Spec()

	gt.NotNil(t, spec)
	gt.V(t, len(spec.FunctionDeclarations)).Equal(1)

	decl := spec.FunctionDeclarations[0]
	gt.Equal(t, decl.Name, "bigquery_run")
	gt.S(t, decl.Description).Contains("natural language")

	// Should have query parameter
	queryParam := decl.Parameters.Properties["query"]
	gt.NotNil(t, queryParam)
	gt.Equal(t, queryParam.Type, genai.TypeString)
}

func TestTool_PromptWithRunbooks(t *testing.T) {
	tmpDir := t.TempDir()
	runbookFile := filepath.Join(tmpDir, "test.sql")

	runbookContent := `-- title: Test Query
-- description: A test query

SELECT 1`

	err := os.WriteFile(runbookFile, []byte(runbookContent), 0644)
	gt.NoError(t, err)

	ctx := context.Background()
	bqTool := bigquery.New()

	// Set flags via reflection (simulating CLI flag parsing)
	flags := bqTool.Flags()
	for _, flag := range flags {
		switch f := flag.(type) {
		case interface{ GetDestination() *string }:
			switch flag.Names()[0] {
			case "bigquery-project":
				*f.GetDestination() = "test-project"
			case "bigquery-runbook-dir":
				*f.GetDestination() = tmpDir
			}
		}
	}

	// Note: Cannot fully test Init without real Gemini and BigQuery clients
	// This test just verifies the flag setup works correctly
	prompt := bqTool.Prompt(ctx)
	// Prompt should be empty before Init is called
	gt.Equal(t, prompt, "")
}

func TestTool_ExecuteInvalidFunction(t *testing.T) {
	ctx := context.Background()
	bqTool := bigquery.New()

	_, err := bqTool.Execute(ctx, genai.FunctionCall{
		Name: "invalid_function",
		Args: map[string]any{},
	})

	gt.Error(t, err)
}

func TestTool_ExecuteMissingQuery(t *testing.T) {
	ctx := context.Background()
	bqTool := bigquery.New()

	resp, err := bqTool.Execute(ctx, genai.FunctionCall{
		Name: "bigquery_run",
		Args: map[string]any{},
	})

	gt.NoError(t, err)
	gt.NotNil(t, resp)

	_, hasError := resp.Response["error"]
	gt.True(t, hasError)
}
