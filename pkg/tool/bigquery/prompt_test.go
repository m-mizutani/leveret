package bigquery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/adapter"
)

func TestPrompt_WithRunBooks(t *testing.T) {
	// Create test runbooks
	tmpDir := t.TempDir()
	runbook1 := `-- title: User Activity Analysis
-- description: Analyze user activity patterns

SELECT user_id, COUNT(*) FROM events GROUP BY user_id`

	err := os.WriteFile(filepath.Join(tmpDir, "user_activity.sql"), []byte(runbook1), 0644)
	gt.NoError(t, err)

	runbook2 := `-- title: Error Detection
-- description: Find errors in logs

SELECT * FROM logs WHERE level = 'ERROR'`

	err = os.WriteFile(filepath.Join(tmpDir, "error_detection.sql"), []byte(runbook2), 0644)
	gt.NoError(t, err)

	// Initialize tool
	bqTool := &Tool{
		project:    "test-project",
		runBookDir: tmpDir,
	}

	// Load runbooks
	runBooks, err := loadRunBooks(tmpDir)
	gt.NoError(t, err)
	bqTool.runBooks = runBooks

	// Generate prompt
	ctx := context.Background()
	prompt := bqTool.Prompt(ctx)

	// Verify prompt contains runbook information
	gt.S(t, prompt).Contains("Available BigQuery runBooks:")
	gt.S(t, prompt).Contains("user_activity")
	gt.S(t, prompt).Contains("User Activity Analysis")
	gt.S(t, prompt).Contains("Analyze user activity patterns")
	gt.S(t, prompt).Contains("error_detection")
	gt.S(t, prompt).Contains("Error Detection")
	gt.S(t, prompt).Contains("Find errors in logs")

	t.Logf("Prompt:\n%s", prompt)
}

func TestPrompt_WithTables(t *testing.T) {
	// Create test tables
	tables := []tableInfo{
		{
			Project:     "proj1",
			Dataset:     "dataset1",
			Table:       "events",
			Description: "User event logs",
		},
		{
			Project:     "proj1",
			Dataset:     "dataset1",
			Table:       "errors",
			Description: "Application error logs",
		},
		{
			Project:     "proj2",
			Dataset:     "dataset2",
			Table:       "metrics",
			Description: "Performance metrics",
		},
	}

	// Initialize tool
	bqTool := &Tool{
		project: "test-project",
		tables:  tables,
	}

	// Generate prompt
	ctx := context.Background()
	prompt := bqTool.Prompt(ctx)

	// Verify prompt contains table information
	gt.S(t, prompt).Contains("Available BigQuery tables:")
	gt.S(t, prompt).Contains("proj1.dataset1.events")
	gt.S(t, prompt).Contains("User event logs")
	gt.S(t, prompt).Contains("proj1.dataset1.errors")
	gt.S(t, prompt).Contains("Application error logs")
	gt.S(t, prompt).Contains("proj2.dataset2.metrics")
	gt.S(t, prompt).Contains("Performance metrics")

	t.Logf("Prompt:\n%s", prompt)
}

func TestPrompt_WithRunBooksAndTables(t *testing.T) {
	// Create test runbooks
	tmpDir := t.TempDir()
	runbook := `-- title: Test Query
-- description: A test query

SELECT 1`

	err := os.WriteFile(filepath.Join(tmpDir, "test.sql"), []byte(runbook), 0644)
	gt.NoError(t, err)

	// Create test tables
	tables := []tableInfo{
		{
			Project:     "proj1",
			Dataset:     "ds1",
			Table:       "tbl1",
			Description: "Test table",
		},
	}

	// Initialize tool
	bqTool := &Tool{
		project:    "test-project",
		runBookDir: tmpDir,
		tables:     tables,
	}

	// Load runbooks
	runBooks, err := loadRunBooks(tmpDir)
	gt.NoError(t, err)
	bqTool.runBooks = runBooks

	// Generate prompt
	ctx := context.Background()
	prompt := bqTool.Prompt(ctx)

	// Verify prompt contains both runbooks and tables
	gt.S(t, prompt).Contains("Available BigQuery runBooks:")
	gt.S(t, prompt).Contains("test")
	gt.S(t, prompt).Contains("Test Query")
	gt.S(t, prompt).Contains("Available BigQuery tables:")
	gt.S(t, prompt).Contains("proj1.ds1.tbl1")
	gt.S(t, prompt).Contains("Test table")

	t.Logf("Prompt:\n%s", prompt)
}

func TestPrompt_Empty(t *testing.T) {
	// Initialize tool with no runbooks or tables
	bqTool := &Tool{
		project: "test-project",
	}

	// Generate prompt
	ctx := context.Background()
	prompt := bqTool.Prompt(ctx)

	// Verify prompt is empty
	gt.Equal(t, prompt, "")
}

func TestPrompt_Integration(t *testing.T) {
	// Create temporary directories
	tmpDir := t.TempDir()
	runbookDir := filepath.Join(tmpDir, "runbooks")
	err := os.Mkdir(runbookDir, 0755)
	gt.NoError(t, err)

	// Create runbook
	runbook := `-- title: Recent Activity
-- description: Get recent user activities

SELECT * FROM events WHERE timestamp > CURRENT_TIMESTAMP() - INTERVAL 1 DAY`

	err = os.WriteFile(filepath.Join(runbookDir, "recent.sql"), []byte(runbook), 0644)
	gt.NoError(t, err)

	// Create config file
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `tables:
  - project: test-project
    dataset: test_dataset
    table: events
    description: Event tracking table
  - project: test-project
    dataset: test_dataset
    table: users
    description: User information table`

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	gt.NoError(t, err)

	// Create and initialize tool
	ctx := context.Background()
	bqTool := &Tool{
		project:    "test-project",
		runBookDir: runbookDir,
		configFile: configFile,
	}

	// Mock BigQuery adapter
	bqTool.bq = &mockBigQuery{}

	// Load runbooks and tables
	runBooks, err := loadRunBooks(runbookDir)
	gt.NoError(t, err)
	bqTool.runBooks = runBooks

	tables, err := loadTableList(configFile)
	gt.NoError(t, err)
	bqTool.tables = tables

	// Generate prompt
	prompt := bqTool.Prompt(ctx)

	// Verify comprehensive prompt
	gt.S(t, prompt).Contains("Available BigQuery runBooks:")
	gt.S(t, prompt).Contains("recent")
	gt.S(t, prompt).Contains("Recent Activity")
	gt.S(t, prompt).Contains("Get recent user activities")

	gt.S(t, prompt).Contains("Available BigQuery tables:")
	gt.S(t, prompt).Contains("test-project.test_dataset.events")
	gt.S(t, prompt).Contains("Event tracking table")
	gt.S(t, prompt).Contains("test-project.test_dataset.users")
	gt.S(t, prompt).Contains("User information table")

	t.Logf("Prompt:\n%s", prompt)
}

// mockBigQuery is a mock implementation for testing
type mockBigQuery struct {
	adapter.BigQuery
}

func (m *mockBigQuery) DryRun(ctx context.Context, query string) (int64, error) {
	return 1024, nil
}

func (m *mockBigQuery) Query(ctx context.Context, query string) (string, error) {
	return "test-job-id", nil
}

func (m *mockBigQuery) GetQueryResult(ctx context.Context, jobID string) ([]map[string]any, error) {
	return []map[string]any{
		{"col1": "value1", "col2": 123},
	}, nil
}
