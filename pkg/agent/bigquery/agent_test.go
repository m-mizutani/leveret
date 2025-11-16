package bigquery

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"google.golang.org/genai"
)

type mockGemini struct {
	adapter.Gemini
	generateFunc func(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

func (m *mockGemini) GenerateContent(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return m.generateFunc(ctx, contents, config)
}

type mockBigQuery struct {
	adapter.BigQuery
	dryRunFunc         func(ctx context.Context, query string) (int64, error)
	queryFunc          func(ctx context.Context, query string) (string, error)
	getQueryResultFunc func(ctx context.Context, jobID string) ([]map[string]any, error)
}

func (m *mockBigQuery) DryRun(ctx context.Context, query string) (int64, error) {
	return m.dryRunFunc(ctx, query)
}

func (m *mockBigQuery) Query(ctx context.Context, query string) (string, error) {
	return m.queryFunc(ctx, query)
}

func (m *mockBigQuery) GetQueryResult(ctx context.Context, jobID string) ([]map[string]any, error) {
	return m.getQueryResultFunc(ctx, jobID)
}

func TestAgent_Execute(t *testing.T) {
	ctx := context.Background()

	// Mock Gemini that returns final text response immediately
	mockGem := &mockGemini{
		generateFunc: func(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Role: genai.RoleModel,
							Parts: []*genai.Part{
								{Text: "Analysis complete. No data found."},
							},
						},
					},
				},
			}, nil
		},
	}

	mockBQ := &mockBigQuery{
		dryRunFunc: func(ctx context.Context, query string) (int64, error) {
			return 1024 * 1024, nil // 1MB
		},
		queryFunc: func(ctx context.Context, query string) (string, error) {
			return "test-job-id", nil
		},
		getQueryResultFunc: func(ctx context.Context, jobID string) ([]map[string]any, error) {
			return []map[string]any{
				{"col1": "value1", "col2": 123},
			}, nil
		},
	}

	agent := NewAgent(mockGem, mockBQ)
	result, err := agent.Execute(ctx, "Find all errors in the logs")

	gt.NoError(t, err)
	gt.S(t, result).Contains("Analysis complete")
}

func TestAgent_ExecuteWithToolCall(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	mockGem := &mockGemini{
		generateFunc: func(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			callCount++
			if callCount == 1 {
				// First call: request tool execution
				return &genai.GenerateContentResponse{
					Candidates: []*genai.Candidate{
						{
							Content: &genai.Content{
								Role: genai.RoleModel,
								Parts: []*genai.Part{
									{
										FunctionCall: &genai.FunctionCall{
											Name: "bigquery_query",
											Args: map[string]any{
												"query": "SELECT * FROM logs LIMIT 10",
											},
										},
									},
								},
							},
						},
					},
				}, nil
			}
			// Second call: return final response
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Role: genai.RoleModel,
							Parts: []*genai.Part{
								{Text: "Found 1 row with test data."},
							},
						},
					},
				},
			}, nil
		},
	}

	mockBQ := &mockBigQuery{
		dryRunFunc: func(ctx context.Context, query string) (int64, error) {
			return 1024 * 1024, nil // 1MB
		},
		queryFunc: func(ctx context.Context, query string) (string, error) {
			return "test-job-id", nil
		},
		getQueryResultFunc: func(ctx context.Context, jobID string) ([]map[string]any, error) {
			return []map[string]any{
				{"col1": "value1", "col2": 123},
			}, nil
		},
	}

	agent := NewAgent(mockGem, mockBQ)
	result, err := agent.Execute(ctx, "Query the logs table")

	gt.NoError(t, err)
	gt.S(t, result).Contains("Found 1 row")
	gt.Equal(t, callCount, 2)
}

func TestAgent_BuildSystemPrompt(t *testing.T) {
	runBooks := map[string]*runBook{
		"test": {
			ID:          "test",
			Title:       "Test Query",
			Description: "A test query",
			Query:       "SELECT 1",
		},
	}

	tables := []tableInfo{
		{
			Project:     "proj1",
			Dataset:     "ds1",
			Table:       "tbl1",
			Description: "Test table",
		},
	}

	agent := NewAgent(nil, nil, WithRunBooks(runBooks), WithTables(tables))
	prompt := agent.buildSystemPrompt()

	gt.S(t, prompt).Contains("Available RunBooks")
	gt.S(t, prompt).Contains("test")
	gt.S(t, prompt).Contains("Test Query")
	gt.S(t, prompt).Contains("A test query")

	gt.S(t, prompt).Contains("Available Tables")
	gt.S(t, prompt).Contains("proj1.ds1.tbl1")
	gt.S(t, prompt).Contains("Test table")
}

func TestAgent_HandleQuery(t *testing.T) {
	ctx := context.Background()

	mockBQ := &mockBigQuery{
		dryRunFunc: func(ctx context.Context, query string) (int64, error) {
			return 100 * 1024 * 1024, nil // 100MB
		},
		queryFunc: func(ctx context.Context, query string) (string, error) {
			return "job-123", nil
		},
		getQueryResultFunc: func(ctx context.Context, jobID string) ([]map[string]any, error) {
			return []map[string]any{
				{"id": 1, "name": "test"},
				{"id": 2, "name": "test2"},
			}, nil
		},
	}

	agent := NewAgent(nil, mockBQ, WithScanLimitMB(1024))

	args := map[string]any{
		"query": "SELECT * FROM events",
	}

	result := agent.handleQuery(ctx, args)

	gt.V(t, result["job_id"]).Equal("job-123")
	gt.V(t, result["total_rows"]).Equal(2)
	gt.V(t, result["scan_size_mb"]).Equal(int64(100))
}

func TestAgent_HandleQueryExceedsLimit(t *testing.T) {
	ctx := context.Background()

	mockBQ := &mockBigQuery{
		dryRunFunc: func(ctx context.Context, query string) (int64, error) {
			return 2000 * 1024 * 1024, nil // 2000MB
		},
	}

	agent := NewAgent(nil, mockBQ, WithScanLimitMB(1024))

	args := map[string]any{
		"query": "SELECT * FROM huge_table",
	}

	result := agent.handleQuery(ctx, args)

	errorMsg, ok := result["error"].(string)
	gt.True(t, ok)
	gt.S(t, errorMsg).Contains("exceeds limit")
}

func TestAgent_HandleGetResult(t *testing.T) {
	ctx := context.Background()

	agent := NewAgent(nil, nil, WithResultLimitRows(100))

	// Pre-populate results
	agent.results["job-123"] = []map[string]any{
		{"id": 1},
		{"id": 2},
		{"id": 3},
		{"id": 4},
		{"id": 5},
	}

	t.Run("with pagination", func(t *testing.T) {
		args := map[string]any{
			"job_id": "job-123",
			"limit":  float64(2),
			"offset": float64(1),
		}

		result := agent.handleGetResult(ctx, args)

		rows, ok := result["rows"].([]map[string]any)
		gt.True(t, ok)
		gt.V(t, len(rows)).Equal(2)
		gt.V(t, rows[0]["id"]).Equal(2)
		gt.V(t, result["has_more"]).Equal(true)
	})

	t.Run("job not found", func(t *testing.T) {
		args := map[string]any{
			"job_id": "nonexistent",
		}

		result := agent.handleGetResult(ctx, args)

		_, hasError := result["error"]
		gt.True(t, hasError)
	})
}

func TestAgent_HandleRunbook(t *testing.T) {
	ctx := context.Background()

	runBooks := map[string]*runBook{
		"test": {
			ID:          "test",
			Title:       "Test Query",
			Description: "A test query",
			Query:       "SELECT 1",
		},
	}

	agent := NewAgent(nil, nil, WithRunBooks(runBooks))

	t.Run("found", func(t *testing.T) {
		args := map[string]any{
			"runbook_id": "test",
		}

		result := agent.handleRunbook(ctx, args)

		gt.V(t, result["id"]).Equal("test")
		gt.V(t, result["title"]).Equal("Test Query")
		gt.V(t, result["query"]).Equal("SELECT 1")
	})

	t.Run("not found", func(t *testing.T) {
		args := map[string]any{
			"runbook_id": "nonexistent",
		}

		result := agent.handleRunbook(ctx, args)

		_, hasError := result["error"]
		gt.True(t, hasError)
	})
}
