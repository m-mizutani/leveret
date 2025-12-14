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
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
	"google.golang.org/genai"
)

//go:embed prompt/system.md
var systemPromptRaw string

// Agent is the BigQuery sub-agent that processes natural language queries
type Agent struct {
	gemini          adapter.Gemini
	bq              adapter.BigQuery
	repo            repository.Repository
	runBooks        map[string]*runBook
	tables          []tableInfo
	scanLimitMB     int64
	resultLimitRows int64
	results         map[string][]map[string]any
	output          io.Writer

	// Session tracking for introspection
	sessionHistory []*genai.Content
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

// WithRepository sets the repository for memory storage
func WithRepository(repo repository.Repository) AgentOption {
	return func(a *Agent) {
		a.repo = repo
	}
}

// Execute processes a natural language query and returns the result
func (a *Agent) Execute(ctx context.Context, query string) (string, error) {
	// Reset session tracking
	a.sessionHistory = []*genai.Content{}

	// (1) Retrieve similar memories if repository is available
	var providedMemories []*model.Memory
	if a.repo != nil {
		// Generate embedding for the query
		embedding, err := a.gemini.Embedding(ctx, query, 768)
		if err != nil {
			if a.output != nil {
				fmt.Fprintf(a.output, "⚠️  Failed to generate embedding for memory search: %v\n", err)
			}
		} else {
			// Search for similar memories (cosine distance threshold 0.8, limit 32)
			memories, err := a.repo.SearchMemories(ctx, embedding, 0.8, 32)
			if err != nil {
				if a.output != nil {
					fmt.Fprintf(a.output, "⚠️  Failed to search memories: %v\n", err)
				}
			} else {
				if a.output != nil {
					fmt.Fprintf(a.output, "🔍 Memory search completed: found %d memories (threshold: 0.8)\n", len(memories))
				}
				if len(memories) > 0 {
					providedMemories = memories
				}
			}
		}
	} else {
		if a.output != nil {
			fmt.Fprintf(a.output, "ℹ️  Repository not available, skipping memory search\n")
		}
	}

	// Build system prompt with context and memories
	systemPrompt := a.buildSystemPrompt(providedMemories)

	// Create initial user message
	contents := []*genai.Content{
		genai.NewContentFromText(query, genai.RoleUser),
	}

	// Build config with tools
	thinkingBudget := int32(0)
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, ""),
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: false,
			ThinkingBudget:  &thinkingBudget,
		},
		Tools: []*genai.Tool{a.internalToolSpec()},
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
		var functionResponses []*genai.Part

		for _, part := range candidate.Content.Parts {
			// Display LLM's intermediate text (subtle format)
			if part.Text != "" && a.output != nil {
				fmt.Fprintf(a.output, "      💭 %s\n", part.Text)
			}

			if part.FunctionCall != nil {
				hasFuncCall = true
				// Execute the internal tool
				funcResp := a.executeInternalTool(ctx, *part.FunctionCall)

				// Collect function response (will be added as single Content later)
				functionResponses = append(functionResponses, &genai.Part{FunctionResponse: funcResp})
			}
		}

		// Add all function responses as a single Content
		if len(functionResponses) > 0 {
			funcRespContent := &genai.Content{
				Role:  genai.RoleUser,
				Parts: functionResponses,
			}
			contents = append(contents, funcRespContent)
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

	// Store session history for introspection
	a.sessionHistory = contents

	// (2) Introspection phase (after execution)
	if a.repo != nil {
		if err := a.runIntrospection(ctx, query, providedMemories, a.sessionHistory); err != nil {
			if a.output != nil {
				fmt.Fprintf(a.output, "⚠️  Introspection failed: %v\n", err)
			}
			// Don't fail the whole execution if introspection fails
		}
	}

	return finalResponse, nil
}

type systemPromptData struct {
	RunBooks []runBook
	Tables   []tableInfo
	Memories []*model.Memory
}

func (a *Agent) buildSystemPrompt(memories []*model.Memory) string {
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
		Memories: memories,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Fallback to raw template if execution fails
		return systemPromptRaw
	}

	// Append memories section if available
	if len(memories) > 0 {
		buf.WriteString("\n\n## Past Knowledge (Memories)\n\n")
		buf.WriteString("The following knowledge was learned from past sessions. Use this information to inform your analysis:\n\n")
		for _, mem := range memories {
			buf.WriteString(fmt.Sprintf("- **Memory ID**: %s\n  **Content**: %s\n\n", mem.ID, mem.Claim))
		}
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

	// Show progress
	if a.output != nil {
		// Show abbreviated query (first 100 chars)
		displayQuery := queryStr
		if len(displayQuery) > 100 {
			displayQuery = displayQuery[:100] + "..."
		}
		fmt.Fprintf(a.output, "🔍 BigQuery実行中: %s\n", displayQuery)
	}

	// Dry run to check scan size
	scanBytes, err := a.bq.DryRun(ctx, queryStr)
	if err != nil {
		errMsg := fmt.Sprintf("dry run failed: %v", err)
		if a.output != nil {
			fmt.Fprintf(a.output, "❌ BigQueryエラー: %s\n", errMsg)
		}
		return map[string]any{"error": errMsg}
	}

	scanMB := scanBytes / (1024 * 1024)
	if scanMB > a.scanLimitMB {
		errMsg := fmt.Sprintf("query scan size (%d MB) exceeds limit (%d MB)", scanMB, a.scanLimitMB)
		if a.output != nil {
			fmt.Fprintf(a.output, "❌ BigQueryエラー: %s\n", errMsg)
		}
		return map[string]any{
			"error":           errMsg,
			"scan_size_bytes": scanBytes,
			"scan_size_mb":    scanMB,
			"limit_mb":        a.scanLimitMB,
		}
	}

	// Execute query
	jobID, err := a.bq.Query(ctx, queryStr)
	if err != nil {
		errMsg := fmt.Sprintf("query execution failed: %v", err)
		if a.output != nil {
			fmt.Fprintf(a.output, "❌ BigQueryエラー: %s\n", errMsg)
		}
		return map[string]any{"error": errMsg}
	}

	// Get and store results
	results, err := a.bq.GetQueryResult(ctx, jobID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get results: %v", err)
		if a.output != nil {
			fmt.Fprintf(a.output, "❌ BigQueryエラー: %s\n", errMsg)
		}
		return map[string]any{"error": errMsg}
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

	// Show progress
	if a.output != nil {
		fmt.Fprintf(a.output, "🔍 BigQueryスキーマ取得中: %s.%s.%s\n", project, datasetID, table)
	}

	metadata, err := a.bq.GetTableMetadata(ctx, project, datasetID, table)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get schema: %v", err)
		if a.output != nil {
			fmt.Fprintf(a.output, "❌ BigQueryエラー: %s\n", errMsg)
		}
		return map[string]any{"error": errMsg}
	}

	// Convert schema to JSON representation
	schemaJSON, err := json.Marshal(metadata.Schema)
	if err != nil {
		errMsg := fmt.Sprintf("failed to marshal schema: %v", err)
		if a.output != nil {
			fmt.Fprintf(a.output, "❌ BigQueryエラー: %s\n", errMsg)
		}
		return map[string]any{"error": errMsg}
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

// runIntrospection performs introspection after execution and updates memories
func (a *Agent) runIntrospection(ctx context.Context, queryText string, providedMemories []*model.Memory, sessionHistory []*genai.Content) error {
	// Run introspection using session history
	if a.output != nil {
		fmt.Fprintf(a.output, "[BigQuery Agent] 🤔 振り返り中...\n")
	}
	result, err := introspect(ctx, a.gemini, queryText, providedMemories, sessionHistory)
	if err != nil {
		return goerr.Wrap(err, "introspection failed")
	}

	// Generate embedding for the query (reuse from earlier or regenerate)
	embedding, err := a.gemini.Embedding(ctx, queryText, 768)
	if err != nil {
		return goerr.Wrap(err, "failed to generate embedding for claims")
	}

	// Build output buffer
	var outputBuf strings.Builder

	// Save new claims as memories
	if len(result.Claims) > 0 {
		outputBuf.WriteString(fmt.Sprintf("\n[BigQuery Agent] 💡 新たな知見を抽出 (%d件):\n", len(result.Claims)))

		for _, claim := range result.Claims {
			memory := &model.Memory{
				ID:        model.NewMemoryID(),
				Claim:     claim.Content,
				QueryText: queryText,
				Embedding: embedding,
				Score:     0.0,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := a.repo.PutMemory(ctx, memory); err != nil {
				outputBuf.WriteString(fmt.Sprintf("  ⚠️  Failed to save memory %s: %v\n", memory.ID, err))
				continue
			}

			outputBuf.WriteString(fmt.Sprintf("  - [%s] %s\n", memory.ID, claim.Content))
		}
	}

	// Update scores for helpful memories
	if len(result.HelpfulMemoryIDs) > 0 {
		outputBuf.WriteString(fmt.Sprintf("\n[BigQuery Agent] 👍 役に立った記憶 (%d件):\n", len(result.HelpfulMemoryIDs)))
		for _, memID := range result.HelpfulMemoryIDs {
			if err := a.repo.UpdateMemoryScore(ctx, model.MemoryID(memID), 1.0); err != nil {
				outputBuf.WriteString(fmt.Sprintf("  ⚠️  Failed to update score for memory %s: %v\n", memID, err))
				continue
			}
			// Find the memory content to display
			var memContent string
			for _, mem := range providedMemories {
				if mem.ID == model.MemoryID(memID) {
					memContent = mem.Claim
					break
				}
			}
			outputBuf.WriteString(fmt.Sprintf("  - [%s] %s (スコア +1.0)\n", memID, memContent))
		}
	}

	// Update scores for harmful memories (those that were incorrect and caused errors)
	if len(result.HarmfulMemoryIDs) > 0 {
		outputBuf.WriteString(fmt.Sprintf("\n[BigQuery Agent] 👎 有害だった記憶 (%d件):\n", len(result.HarmfulMemoryIDs)))
		for _, memID := range result.HarmfulMemoryIDs {
			if err := a.repo.UpdateMemoryScore(ctx, model.MemoryID(memID), -1.0); err != nil {
				outputBuf.WriteString(fmt.Sprintf("  ⚠️  Failed to update score for harmful memory %s: %v\n", memID, err))
				continue
			}
			// Find the memory content to display
			var memContent string
			for _, mem := range providedMemories {
				if mem.ID == model.MemoryID(memID) {
					memContent = mem.Claim
					break
				}
			}
			outputBuf.WriteString(fmt.Sprintf("  - [%s] %s (スコア -1.0)\n", memID, memContent))
		}
	}

	// Output all at once
	if a.output != nil && outputBuf.Len() > 0 {
		fmt.Fprint(a.output, outputBuf.String())
	}

	// Delete memories below threshold (-3.0)
	if err := a.repo.DeleteMemoriesBelowScore(ctx, -3.0); err != nil {
		if a.output != nil {
			fmt.Fprintf(a.output, "⚠️  Failed to delete low-score memories: %v\n", err)
		}
	}

	return nil
}
