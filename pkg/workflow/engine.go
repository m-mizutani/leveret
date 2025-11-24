package workflow

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown/print"
	"google.golang.org/genai"
)

//go:embed prompt/enrich.md
var enrichPromptRaw string

var enrichPromptTmpl = template.Must(template.New("enrich").Parse(enrichPromptRaw))

// regoPrintHook implements print.Hook interface for Rego print() statements
type regoPrintHook struct{}

func (h *regoPrintHook) Print(ctx print.Context, message string) error {
	fmt.Printf("   [Rego] %s\n", message)
	return nil
}

// Engine is the workflow engine that orchestrates the three phases
type Engine struct {
	ingestPolicy *rego.PreparedEvalQuery
	enrichPolicy *rego.PreparedEvalQuery
	triagePolicy *rego.PreparedEvalQuery

	gemini   adapter.Gemini
	registry *tool.Registry
}

// New creates a new workflow engine
func New(ctx context.Context, policyDir string, gemini adapter.Gemini, registry *tool.Registry) (*Engine, error) {
	ingest, enrich, triage, err := loadPolicies(ctx, policyDir)
	if err != nil {
		return nil, err
	}

	return &Engine{
		ingestPolicy: ingest,
		enrichPolicy: enrich,
		triagePolicy: triage,
		gemini:       gemini,
		registry:     registry,
	}, nil
}

// Execute runs the workflow on the raw alert data
func (e *Engine) Execute(ctx context.Context, rawData any) ([]*WorkflowResult, error) {
	// Phase 1: Ingest
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📥 INGEST PHASE\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	ingestResult, err := e.runIngest(ctx, rawData)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to run ingest phase")
	}

	// If no alerts generated, return empty result
	if len(ingestResult.Alert) == 0 {
		fmt.Printf("❌ No alerts generated (rejected by policy)\n\n")
		return nil, nil
	}

	fmt.Printf("✅ Generated %d alert(s)\n", len(ingestResult.Alert))
	for i, alert := range ingestResult.Alert {
		fmt.Printf("   %d. %s\n", i+1, alert.Title)
	}
	fmt.Printf("\n")

	// Process each alert
	results := make([]*WorkflowResult, 0, len(ingestResult.Alert))
	for i, alert := range ingestResult.Alert {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("📋 ALERT %d/%d: %s\n", i+1, len(ingestResult.Alert), alert.Title)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		result, err := e.processAlert(ctx, alert, rawData)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to process alert")
		}
		results = append(results, result)
		fmt.Printf("\n")
	}

	return results, nil
}

func (e *Engine) processAlert(ctx context.Context, ingestedAlert *IngestedAlert, rawData any) (*WorkflowResult, error) {
	// Convert IngestedAlert to model.Alert
	alert := &model.Alert{
		Title:       ingestedAlert.Title,
		Description: ingestedAlert.Description,
		Data:        rawData,
		Attributes:  ingestedAlert.Attributes,
	}

	result := &WorkflowResult{
		Alert: alert,
	}

	// Phase 2: Enrich
	if e.enrichPolicy != nil {
		fmt.Printf("\n🔍 ENRICH PHASE\n")
		enrichResult, enrichExecution, err := e.runEnrich(ctx, alert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to run enrich phase")
		}
		if len(enrichResult.Prompt) == 0 {
			fmt.Printf("   ℹ️  No enrichment prompts generated\n")
		} else {
			fmt.Printf("   ✅ Executed %d enrichment task(s)\n", len(enrichResult.Prompt))
			for i, exec := range enrichExecution.Result {
				fmt.Printf("      %d. %s: ", i+1, exec.ID)
				if len(exec.Result) > 60 {
					fmt.Printf("%s...\n", exec.Result[:60])
				} else {
					fmt.Printf("%s\n", exec.Result)
				}
			}
		}
		result.EnrichResult = enrichResult
		result.EnrichExecution = enrichExecution
	}

	// Phase 3: Triage
	if e.triagePolicy != nil {
		fmt.Printf("\n⚖️  TRIAGE PHASE\n")
		triageResult, err := e.runTriage(ctx, alert, result.EnrichExecution)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to run triage phase")
		}

		// Display triage decision with appropriate emoji
		actionEmoji := "📌"
		switch triageResult.Action {
		case "accept":
			actionEmoji = "✅"
		case "discard":
			actionEmoji = "🗑️"
		case "notify":
			actionEmoji = "📢"
		}

		severityEmoji := "📊"
		switch triageResult.Severity {
		case "critical":
			severityEmoji = "🔴"
		case "high":
			severityEmoji = "🟠"
		case "medium":
			severityEmoji = "🟡"
		case "low":
			severityEmoji = "🟢"
		case "info":
			severityEmoji = "ℹ️"
		}

		fmt.Printf("   %s Action: %s\n", actionEmoji, triageResult.Action)
		fmt.Printf("   %s Severity: %s\n", severityEmoji, triageResult.Severity)
		if triageResult.Note != "" {
			fmt.Printf("   📝 Note: %s\n", triageResult.Note)
		}
		result.Triage = triageResult
	}

	return result, nil
}

func (e *Engine) runIngest(ctx context.Context, rawData any) (*IngestResult, error) {
	if e.ingestPolicy == nil {
		// No policy, accept all with empty result
		return &IngestResult{Alert: nil}, nil
	}

	rs, err := e.ingestPolicy.Eval(ctx, rego.EvalInput(rawData), rego.EvalPrintHook(&regoPrintHook{}))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to evaluate ingest policy")
	}

	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return &IngestResult{Alert: nil}, nil
	}

	// Parse result
	data := rs[0].Expressions[0].Value.(map[string]any)
	alertData, ok := data["alert"]
	if !ok {
		return &IngestResult{Alert: nil}, nil
	}

	alerts, ok := alertData.([]any)
	if !ok {
		return nil, goerr.New("invalid ingest result: alert is not an array")
	}

	result := &IngestResult{
		Alert: make([]*IngestedAlert, 0, len(alerts)),
	}

	for _, a := range alerts {
		alertMap, ok := a.(map[string]any)
		if !ok {
			return nil, goerr.New("invalid alert in ingest result")
		}

		alert := &IngestedAlert{
			Title:       getString(alertMap, "title"),
			Description: getString(alertMap, "description"),
			Attributes:  parseAttributes(alertMap["attributes"]),
		}
		result.Alert = append(result.Alert, alert)
	}

	return result, nil
}

func (e *Engine) runEnrich(ctx context.Context, alert *model.Alert) (*EnrichResult, *EnrichExecution, error) {
	if e.enrichPolicy == nil {
		return &EnrichResult{}, &EnrichExecution{}, nil
	}

	// Prepare input for enrich policy
	input := map[string]any{
		"id":          alert.ID,
		"title":       alert.Title,
		"description": alert.Description,
		"attributes":  alert.Attributes,
	}

	rs, err := e.enrichPolicy.Eval(ctx, rego.EvalInput(input), rego.EvalPrintHook(&regoPrintHook{}))
	if err != nil {
		return nil, nil, goerr.Wrap(err, "failed to evaluate enrich policy")
	}

	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return &EnrichResult{}, &EnrichExecution{}, nil
	}

	// Parse result
	data := rs[0].Expressions[0].Value.(map[string]any)
	promptData, ok := data["prompt"]
	if !ok {
		return &EnrichResult{}, &EnrichExecution{}, nil
	}

	prompts, ok := promptData.([]any)
	if !ok {
		return nil, nil, goerr.New("invalid enrich result: prompt is not an array")
	}

	enrichResult := &EnrichResult{
		Prompt: make([]AgentPrompt, 0, len(prompts)),
	}

	for _, p := range prompts {
		promptMap, ok := p.(map[string]any)
		if !ok {
			return nil, nil, goerr.New("invalid prompt in enrich result")
		}

		prompt := AgentPrompt{
			ID:      getString(promptMap, "id"),
			Content: getString(promptMap, "content"),
			Format:  getString(promptMap, "format"),
		}
		enrichResult.Prompt = append(enrichResult.Prompt, prompt)
	}

	// Execute LLM agents for each prompt
	enrichExecution := &EnrichExecution{
		Result: make([]EnrichExecutionResult, 0, len(enrichResult.Prompt)),
	}

	for i, prompt := range enrichResult.Prompt {
		fmt.Printf("   🤖 Task %d/%d: %s\n", i+1, len(enrichResult.Prompt), prompt.ID)
		result, err := e.executePrompt(ctx, prompt, alert)
		if err != nil {
			return nil, nil, goerr.Wrap(err, "failed to execute prompt", goerr.Value("prompt_id", prompt.ID))
		}
		enrichExecution.Result = append(enrichExecution.Result, EnrichExecutionResult{
			ID:     prompt.ID,
			Result: result,
		})
	}

	return enrichResult, enrichExecution, nil
}

// executePrompt executes a single prompt using LLM with tools
func (e *Engine) executePrompt(ctx context.Context, prompt AgentPrompt, alert *model.Alert) (string, error) {
	// Marshal alert data
	alertDataBytes, err := json.MarshalIndent(alert.Data, "", "  ")
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal alert data")
	}
	alertDataJSON := string(alertDataBytes)

	// Build system instruction using template
	var buf bytes.Buffer
	if err := enrichPromptTmpl.Execute(&buf, map[string]any{
		"PromptContent":  prompt.Content,
		"Alert":          alert,
		"AlertDataJSON":  alertDataJSON,
	}); err != nil {
		return "", goerr.Wrap(err, "failed to execute enrich prompt template")
	}
	systemInstruction := buf.String()

	// Create content for the request
	contents := []*genai.Content{
		genai.NewContentFromText("Execute the task described in the system instruction.", genai.RoleUser),
	}

	// Build config with system instruction and tools
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, ""),
	}

	// Add tools from registry if available
	if e.registry != nil {
		config.Tools = e.registry.Specs()
	}

	// Tool Call loop: keep generating until no more function calls
	const maxIterations = 32
	var finalResult string

	for i := 0; i < maxIterations; i++ {
		resp, err := e.gemini.GenerateContent(ctx, contents, config)
		if err != nil {
			return "", goerr.Wrap(err, "failed to generate content")
		}

		// Check if response contains function calls
		hasFunctionCall := false
		for _, candidate := range resp.Candidates {
			if candidate.Content == nil {
				continue
			}

			// Add assistant response to history
			contents = append(contents, candidate.Content)

			for _, part := range candidate.Content.Parts {
				// Collect text responses
				if part.Text != "" {
					finalResult = part.Text
				}

				if part.FunctionCall != nil {
					hasFunctionCall = true
					// Execute the tool
					funcResp, execErr := e.executeTool(ctx, *part.FunctionCall)
					if execErr != nil {
						// Create error response
						funcResp = &genai.FunctionResponse{
							Name:     part.FunctionCall.Name,
							Response: map[string]any{"error": execErr.Error()},
						}
					}

					// Add function response to history
					funcRespContent := &genai.Content{
						Role:  genai.RoleUser,
						Parts: []*genai.Part{{FunctionResponse: funcResp}},
					}
					contents = append(contents, funcRespContent)
				}
			}
		}

		// If no function call, we're done
		if !hasFunctionCall {
			break
		}
	}

	// If format is JSON, validate and clean the result
	if prompt.Format == "json" {
		finalResult = cleanJSONResponse(finalResult)
	}

	return finalResult, nil
}

// executeTool executes a tool via registry
func (e *Engine) executeTool(ctx context.Context, funcCall genai.FunctionCall) (*genai.FunctionResponse, error) {
	if e.registry == nil {
		return nil, goerr.New("tool registry not available")
	}

	fmt.Printf("      🔧 Tool: %s\n", funcCall.Name)

	// Execute the tool via registry
	resp, err := e.registry.Execute(ctx, funcCall)
	if err != nil {
		fmt.Printf("      ❌ Tool execution failed: %v\n", err)
		return nil, goerr.Wrap(err, "tool execution failed")
	}

	// Check if response contains error
	if errMsg, ok := resp.Response["error"].(string); ok {
		fmt.Printf("         ⚠️  Error: %s\n", errMsg)
		return resp, nil
	}

	fmt.Printf("         ✓ Success\n")
	return resp, nil
}

// cleanJSONResponse removes markdown code blocks and extracts pure JSON
func cleanJSONResponse(response string) string {
	// Remove markdown code blocks if present
	response = removeMarkdownCodeBlocks(response)

	// Try to find JSON object or array
	if jsonStart := findJSONStart(response); jsonStart >= 0 {
		response = response[jsonStart:]
		if jsonEnd := findJSONEnd(response); jsonEnd >= 0 {
			response = response[:jsonEnd+1]
		}
	}

	return response
}

func removeMarkdownCodeBlocks(s string) string {
	// Remove ```json ... ``` blocks
	start := 0
	for {
		idx := findSubstring(s[start:], "```")
		if idx < 0 {
			break
		}
		idx += start
		// Find end of code block
		endIdx := findSubstring(s[idx+3:], "```")
		if endIdx < 0 {
			break
		}
		endIdx += idx + 3

		// Extract content between markers
		content := s[idx+3 : endIdx]
		// Remove language identifier (e.g., "json")
		if newlineIdx := findSubstring(content, "\n"); newlineIdx >= 0 {
			content = content[newlineIdx+1:]
		}

		// Replace the code block with just the content
		s = s[:idx] + content + s[endIdx+3:]
		start = idx + len(content)
	}
	return s
}

func findJSONStart(s string) int {
	for i, c := range s {
		if c == '{' || c == '[' {
			return i
		}
	}
	return -1
}

func findJSONEnd(s string) int {
	depth := 0
	inString := false
	escape := false

	for i, c := range s {
		if escape {
			escape = false
			continue
		}

		if c == '\\' {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == '{' || c == '[' {
			depth++
		} else if c == '}' || c == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func (e *Engine) runTriage(ctx context.Context, alert *model.Alert, enrichExecution *EnrichExecution) (*TriageResult, error) {
	if e.triagePolicy == nil {
		// Default behavior
		return &TriageResult{
			Action:   "accept",
			Severity: "medium",
			Note:     "",
		}, nil
	}

	// Prepare input for triage policy
	var enrichResults []map[string]any
	if enrichExecution != nil {
		enrichResults = make([]map[string]any, 0, len(enrichExecution.Result))
		for _, r := range enrichExecution.Result {
			enrichResults = append(enrichResults, map[string]any{
				"id":     r.ID,
				"result": r.Result,
			})
		}
	} else {
		enrichResults = make([]map[string]any, 0)
	}

	input := map[string]any{
		"alert": map[string]any{
			"id":          alert.ID,
			"title":       alert.Title,
			"description": alert.Description,
			"attributes":  alert.Attributes,
		},
		"enrich": enrichResults,
	}

	rs, err := e.triagePolicy.Eval(ctx, rego.EvalInput(input), rego.EvalPrintHook(&regoPrintHook{}))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to evaluate triage policy")
	}

	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return &TriageResult{
			Action:   "accept",
			Severity: "medium",
			Note:     "",
		}, nil
	}

	// Parse result
	data := rs[0].Expressions[0].Value.(map[string]any)

	result := &TriageResult{
		Action:   getString(data, "action"),
		Severity: getString(data, "severity"),
		Note:     getString(data, "note"),
	}

	return result, nil
}

// Helper functions
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func parseAttributes(data any) []*model.Attribute {
	if data == nil {
		return nil
	}

	attrs, ok := data.([]any)
	if !ok {
		return nil
	}

	result := make([]*model.Attribute, 0, len(attrs))
	for _, a := range attrs {
		attrMap, ok := a.(map[string]any)
		if !ok {
			continue
		}

		attr := &model.Attribute{
			Key:   getString(attrMap, "key"),
			Value: getString(attrMap, "value"),
			Type:  model.AttributeType(getString(attrMap, "type")),
		}
		result = append(result, attr)
	}

	return result
}
