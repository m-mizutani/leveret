package bigquery

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"google.golang.org/genai"
)

//go:embed prompt/system.md
var systemPromptRaw string

// Agent is the BigQuery sub-agent that processes natural language queries
type Agent struct {
	gemini          adapter.Gemini
	bq              adapter.BigQuery
	runBooks        map[string]*runBook
	tables          []tableInfo
	scanLimitMB     int64
	resultLimitRows int64
	results         map[string][]map[string]any
	output          io.Writer
}

// NewAgent creates a new BigQuery agent
func NewAgent(gemini adapter.Gemini, bq adapter.BigQuery, opts ...AgentOption) *Agent {
	a := &Agent{
		gemini:          gemini,
		bq:              bq,
		runBooks:        make(map[string]*runBook),
		results:         make(map[string][]map[string]any),
		scanLimitMB:     1024,
		resultLimitRows: 1000,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// AgentOption is a functional option for Agent
type AgentOption func(*Agent)

// WithRunBooks sets the runBooks for the agent
func WithRunBooks(runBooks map[string]*runBook) AgentOption {
	return func(a *Agent) {
		a.runBooks = runBooks
	}
}

// WithTables sets the tables for the agent
func WithTables(tables []tableInfo) AgentOption {
	return func(a *Agent) {
		a.tables = tables
	}
}

// WithScanLimitMB sets the scan limit in MB
func WithScanLimitMB(limit int64) AgentOption {
	return func(a *Agent) {
		a.scanLimitMB = limit
	}
}

// WithResultLimitRows sets the result limit rows
func WithResultLimitRows(limit int64) AgentOption {
	return func(a *Agent) {
		a.resultLimitRows = limit
	}
}

// WithOutput sets the output writer for status messages
func WithOutput(w io.Writer) AgentOption {
	return func(a *Agent) {
		a.output = w
	}
}

// Execute processes a natural language query and returns the result
func (a *Agent) Execute(ctx context.Context, query string) (string, error) {
	// Build system prompt with context
	systemPrompt := a.buildSystemPrompt()

	// Create initial user message
	contents := []*genai.Content{
		genai.NewContentFromText(query, genai.RoleUser),
	}

	// Build config with tools
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, ""),
		Tools:             []*genai.Tool{a.internalToolSpec()},
	}

	// Tool Call loop
	const maxIterations = 16
	var finalResponse string

	for i := 0; i < maxIterations; i++ {
		resp, err := a.gemini.GenerateContent(ctx, contents, config)
		if err != nil {
			return "", goerr.Wrap(err, "failed to generate content")
		}

		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			return "", goerr.New("empty response from Gemini")
		}

		candidate := resp.Candidates[0]
		contents = append(contents, candidate.Content)

		// Check for function calls
		hasFuncCall := false
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				hasFuncCall = true
				// Execute the internal tool
				funcResp := a.executeInternalTool(ctx, *part.FunctionCall)

				// Add function response to contents
				funcRespContent := &genai.Content{
					Role:  genai.RoleUser,
					Parts: []*genai.Part{{FunctionResponse: funcResp}},
				}
				contents = append(contents, funcRespContent)
			}
		}

		// If no function call, extract final text response
		if !hasFuncCall {
			var textParts []string
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					textParts = append(textParts, part.Text)
				}
			}
			finalResponse = strings.Join(textParts, "\n")
			break
		}
	}

	return finalResponse, nil
}

type systemPromptData struct {
	RunBooks []runBook
	Tables   []tableInfo
}

func (a *Agent) buildSystemPrompt() string {
	tmpl, err := template.New("system").Parse(systemPromptRaw)
	if err != nil {
		// Fallback to raw template if parsing fails
		return systemPromptRaw
	}

	// Convert map to slice for template
	runBookList := make([]runBook, 0, len(a.runBooks))
	for _, rb := range a.runBooks {
		runBookList = append(runBookList, *rb)
	}

	data := systemPromptData{
		RunBooks: runBookList,
		Tables:   a.tables,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Fallback to raw template if execution fails
		return systemPromptRaw
	}

	return buf.String()
}

func (a *Agent) internalToolSpec() *genai.Tool {
	declarations := []*genai.FunctionDeclaration{
		{
			Name:        "bigquery_query",
			Description: fmt.Sprintf("Execute a BigQuery SQL query. Validated for scan limit (max: %d MB).", a.scanLimitMB),
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"query": {
						Type:        genai.TypeString,
						Description: "SQL query to execute",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "bigquery_get_result",
			Description: "Get results from a previously executed BigQuery job",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"job_id": {
						Type:        genai.TypeString,
						Description: "Job ID returned from bigquery_query",
					},
					"limit": {
						Type:        genai.TypeInteger,
						Description: fmt.Sprintf("Maximum rows to return (default: 100, max: %d)", a.resultLimitRows),
					},
					"offset": {
						Type:        genai.TypeInteger,
						Description: "Rows to skip for pagination (default: 0)",
					},
				},
				Required: []string{"job_id"},
			},
		},
		{
			Name:        "bigquery_schema",
			Description: "Get schema information for a BigQuery table",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"project": {
						Type:        genai.TypeString,
						Description: "Google Cloud project ID",
					},
					"dataset_id": {
						Type:        genai.TypeString,
						Description: "BigQuery dataset ID",
					},
					"table": {
						Type:        genai.TypeString,
						Description: "BigQuery table name",
					},
				},
				Required: []string{"project", "dataset_id", "table"},
			},
		},
	}

	// Add bigquery_runbook if runBooks are loaded
	if len(a.runBooks) > 0 {
		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:        "bigquery_runbook",
			Description: "Get SQL query from runBook by ID",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"runbook_id": {
						Type:        genai.TypeString,
						Description: "RunBook ID to retrieve",
					},
				},
				Required: []string{"runbook_id"},
			},
		})
	}

	return &genai.Tool{
		FunctionDeclarations: declarations,
	}
}

func (a *Agent) executeInternalTool(ctx context.Context, fc genai.FunctionCall) *genai.FunctionResponse {
	var response map[string]any

	switch fc.Name {
	case "bigquery_query":
		response = a.handleQuery(ctx, fc.Args)
	case "bigquery_get_result":
		response = a.handleGetResult(ctx, fc.Args)
	case "bigquery_schema":
		response = a.handleSchema(ctx, fc.Args)
	case "bigquery_runbook":
		response = a.handleRunbook(ctx, fc.Args)
	default:
		response = map[string]any{"error": fmt.Sprintf("unknown function: %s", fc.Name)}
	}

	return &genai.FunctionResponse{
		Name:     fc.Name,
		Response: response,
	}
}

func (a *Agent) handleQuery(ctx context.Context, args map[string]any) map[string]any {
	queryStr, ok := args["query"].(string)
	if !ok {
		return map[string]any{"error": "query parameter is required"}
	}

	// Dry run to check scan size
	scanBytes, err := a.bq.DryRun(ctx, queryStr)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("dry run failed: %v", err)}
	}

	scanMB := scanBytes / (1024 * 1024)
	if scanMB > a.scanLimitMB {
		return map[string]any{
			"error":           fmt.Sprintf("query scan size (%d MB) exceeds limit (%d MB)", scanMB, a.scanLimitMB),
			"scan_size_bytes": scanBytes,
			"scan_size_mb":    scanMB,
			"limit_mb":        a.scanLimitMB,
		}
	}

	// Execute query
	jobID, err := a.bq.Query(ctx, queryStr)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("query execution failed: %v", err)}
	}

	// Get and store results
	results, err := a.bq.GetQueryResult(ctx, jobID)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("failed to get results: %v", err)}
	}

	a.results[jobID] = results

	// Output query execution status
	if a.output != nil {
		fmt.Fprintf(a.output, "  📊 BigQuery: %d rows, %d MB scanned\n", len(results), scanMB)
		fmt.Fprintf(a.output, "  Query: %s\n", queryStr)
	}

	return map[string]any{
		"job_id":          jobID,
		"total_rows":      len(results),
		"scan_size_bytes": scanBytes,
		"scan_size_mb":    scanMB,
	}
}

func (a *Agent) handleGetResult(ctx context.Context, args map[string]any) map[string]any {
	jobID, ok := args["job_id"].(string)
	if !ok {
		return map[string]any{"error": "job_id parameter is required"}
	}

	results, exists := a.results[jobID]
	if !exists {
		return map[string]any{"error": fmt.Sprintf("job_id %s not found", jobID)}
	}

	// Parse pagination parameters
	limit := int64(100)
	if l, ok := args["limit"].(float64); ok {
		limit = int64(l)
	}
	if limit > a.resultLimitRows {
		limit = a.resultLimitRows
	}

	offset := int64(0)
	if o, ok := args["offset"].(float64); ok {
		offset = int64(o)
	}

	// Apply pagination
	totalRows := int64(len(results))
	if offset >= totalRows {
		return map[string]any{
			"rows":       []map[string]any{},
			"total_rows": totalRows,
			"offset":     offset,
			"limit":      limit,
			"has_more":   false,
		}
	}

	end := offset + limit
	if end > totalRows {
		end = totalRows
	}

	paginatedResults := results[offset:end]

	return map[string]any{
		"rows":       paginatedResults,
		"total_rows": totalRows,
		"offset":     offset,
		"limit":      limit,
		"has_more":   end < totalRows,
	}
}

func (a *Agent) handleSchema(ctx context.Context, args map[string]any) map[string]any {
	project, ok := args["project"].(string)
	if !ok {
		return map[string]any{"error": "project parameter is required"}
	}
	datasetID, ok := args["dataset_id"].(string)
	if !ok {
		return map[string]any{"error": "dataset_id parameter is required"}
	}
	table, ok := args["table"].(string)
	if !ok {
		return map[string]any{"error": "table parameter is required"}
	}

	metadata, err := a.bq.GetTableMetadata(ctx, project, datasetID, table)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("failed to get schema: %v", err)}
	}

	// Convert schema to JSON representation
	schemaJSON, err := json.Marshal(metadata.Schema)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("failed to marshal schema: %v", err)}
	}

	return map[string]any{
		"schema": string(schemaJSON),
	}
}

func (a *Agent) handleRunbook(ctx context.Context, args map[string]any) map[string]any {
	runbookID, ok := args["runbook_id"].(string)
	if !ok {
		return map[string]any{"error": "runbook_id parameter is required"}
	}

	rb, exists := a.runBooks[runbookID]
	if !exists {
		return map[string]any{"error": fmt.Sprintf("runbook %s not found", runbookID)}
	}

	return map[string]any{
		"id":          rb.ID,
		"title":       rb.Title,
		"description": rb.Description,
		"query":       rb.Query,
	}
}
