package bigquery

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
	"google.golang.org/genai"
)

//go:embed prompt/introspect.md
var introspectPromptRaw string

// introspectionResult contains the output of introspection analysis
type introspectionResult struct {
	Claims           []claim  `json:"claims"`
	HelpfulMemoryIDs []string `json:"helpful_memory_ids"`
	HarmfulMemoryIDs []string `json:"harmful_memory_ids"`
}

type claim struct {
	Content string `json:"content"`
}

type introspectPromptData struct {
	ProvidedMemories []*model.Memory
	QueryText        string
}

// introspect analyzes a completed session and extracts learnings using the full conversation history
func introspect(ctx context.Context, gemini adapter.Gemini, queryText string, providedMemories []*model.Memory, sessionHistory []*genai.Content) (*introspectionResult, error) {
	// Build introspection system instruction using template
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	tmpl, err := template.New("introspect").Funcs(funcMap).Parse(introspectPromptRaw)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to parse introspection template")
	}

	data := introspectPromptData{
		ProvidedMemories: providedMemories,
		QueryText:        queryText,
	}

	var systemPrompt bytes.Buffer
	if err := tmpl.Execute(&systemPrompt, data); err != nil {
		return nil, goerr.Wrap(err, "failed to execute introspection template")
	}

	// Build introspection contents: session history + introspection request
	introspectionContents := make([]*genai.Content, 0, len(sessionHistory)+1)

	// Add all session history (user queries, assistant responses, tool calls, tool results)
	introspectionContents = append(introspectionContents, sessionHistory...)

	// Add introspection request as final user message
	introspectionContents = append(introspectionContents, genai.NewContentFromText(
		"Please analyze the above session execution and extract learnings according to the instructions in the system prompt.",
		genai.RoleUser,
	))

	// Build config with JSON schema for structured output
	thinkingBudget := int32(0)
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt.String(), ""),
		ResponseMIMEType:  "application/json",
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: false,
			ThinkingBudget:  &thinkingBudget,
		},
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"claims": {
					Type:        genai.TypeArray,
					Description: "Extracted learnings from this session",
					Items: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"content": {
								Type:        genai.TypeString,
								Description: "The learning content",
							},
						},
						Required: []string{"content"},
					},
				},
				"helpful_memory_ids": {
					Type:        genai.TypeArray,
					Description: "IDs of memories that were actually used and contributed to correct results",
					Items: &genai.Schema{
						Type: genai.TypeString,
					},
				},
				"harmful_memory_ids": {
					Type:        genai.TypeArray,
					Description: "IDs of memories that were clearly incorrect and caused errors or wasted effort. Do NOT include memories that were simply not used.",
					Items: &genai.Schema{
						Type: genai.TypeString,
					},
				},
			},
			Required: []string{"claims", "helpful_memory_ids", "harmful_memory_ids"},
		},
	}

	// Generate introspection
	resp, err := gemini.GenerateContent(ctx, introspectionContents, config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate introspection")
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, goerr.New("invalid response structure from gemini")
	}

	rawJSON := resp.Candidates[0].Content.Parts[0].Text

	// Parse JSON response
	var result introspectionResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		fmt.Printf("❌ Introspection JSON parse error: %v\n", err)
		fmt.Printf("   Received JSON:\n%s\n", rawJSON)
		return nil, goerr.Wrap(err, "failed to unmarshal introspection JSON", goerr.V("json", rawJSON))
	}

	return &result, nil
}
