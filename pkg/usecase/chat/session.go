package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
	"github.com/m-mizutani/leveret/pkg/tool"
	"google.golang.org/genai"
)

// Session manages an interactive chat session for alert analysis
type Session struct {
	repo     repository.Repository
	gemini   adapter.Gemini
	storage  adapter.Storage
	registry *tool.Registry

	alertID model.AlertID
	alert   *model.Alert
	history *model.History
}

// NewInput contains parameters for creating a new chat session
type NewInput struct {
	Repo      repository.Repository
	Gemini    adapter.Gemini
	Storage   adapter.Storage
	Registry  *tool.Registry
	AlertID   model.AlertID
	HistoryID *model.HistoryID // Optional: specify to continue existing conversation
}

func New(ctx context.Context, input NewInput) (*Session, error) {
	alert, err := input.Repo.GetAlert(ctx, input.AlertID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alert")
	}

	var history *model.History
	if input.HistoryID != nil {
		// Load existing history
		history, err = loadHistory(ctx, input.Repo, input.Storage, *input.HistoryID)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to load history")
		}
	} else {
		// Create new history
		history = &model.History{}
	}

	return &Session{
		repo:     input.Repo,
		gemini:   input.Gemini,
		storage:  input.Storage,
		registry: input.Registry,

		alertID: input.AlertID,
		alert:   alert,
		history: history,
	}, nil
}

func (s *Session) Send(ctx context.Context, message string) (*genai.GenerateContentResponse, error) {
	// Generate title from first user input if this is a new history
	if len(s.history.Contents) == 0 {
		title, err := generateTitle(ctx, s.gemini, message)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to generate title")
		}
		s.history.Title = title
	}

	// Build system prompt with alert data
	alertData, err := json.MarshalIndent(s.alert.Data, "", "  ")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal alert data")
	}

	systemPrompt := "You are a helpful assistant. When asked about the alert, refer to the following data:\n\nAlert Data:\n" + string(alertData)

	// Add tool-specific prompts
	if s.registry != nil {
		if toolPrompts := s.registry.Prompts(ctx); toolPrompts != "" {
			systemPrompt += "\n\n" + toolPrompts
		}
	}

	// Add user message to history
	userContent := genai.NewContentFromText(message, genai.RoleUser)
	s.history.Contents = append(s.history.Contents, userContent)

	// Build config with system instruction and tools
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, ""),
	}

	// Add tools from registry if available
	if s.registry != nil {
		config.Tools = s.registry.Specs()
	}

	// Tool Call loop: keep generating until no more function calls
	const maxIterations = 10
	var finalResp *genai.GenerateContentResponse

	for i := 0; i < maxIterations; i++ {
		resp, err := s.gemini.GenerateContent(ctx, s.history.Contents, config)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to generate content")
		}

		finalResp = resp

		// Check if response contains function calls
		if !hasFunctionCall(resp) {
			// No function call, this is the final response
			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				s.history.Contents = append(s.history.Contents, resp.Candidates[0].Content)
			}
			break
		}

		// Extract and execute function calls
		for _, candidate := range resp.Candidates {
			if candidate.Content == nil {
				continue
			}

			// Add function call content to history
			s.history.Contents = append(s.history.Contents, candidate.Content)

			for _, part := range candidate.Content.Parts {
				if part.FunctionCall == nil {
					continue
				}

				// Execute the tool
				funcResp, err := s.executeTool(ctx, *part.FunctionCall)
				if err != nil {
					// Create error response
					funcResp = &genai.FunctionResponse{
						Name:     part.FunctionCall.Name,
						Response: map[string]any{"error": err.Error()},
					}
				}

				// Add function response to history
				funcRespContent := &genai.Content{
					Role:  genai.RoleUser,
					Parts: []*genai.Part{{FunctionResponse: funcResp}},
				}
				s.history.Contents = append(s.history.Contents, funcRespContent)
			}
		}
	}

	// Save history to Cloud Storage and repository
	if err := saveHistory(ctx, s.repo, s.storage, s.alertID, s.history); err != nil {
		return nil, goerr.Wrap(err, "failed to save history")
	}

	return finalResp, nil
}

func hasFunctionCall(resp *genai.GenerateContentResponse) bool {
	for _, candidate := range resp.Candidates {
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part.FunctionCall != nil {
					return true
				}
			}
		}
	}
	return false
}

func (s *Session) executeTool(ctx context.Context, funcCall genai.FunctionCall) (*genai.FunctionResponse, error) {
	if s.registry == nil {
		return nil, goerr.New("tool registry not available")
	}

	// Display tool call information
	fmt.Printf("\n🔧 Calling tool: %s\n", funcCall.Name)
	if funcCall.Args != nil {
		argsJSON, _ := json.MarshalIndent(funcCall.Args, "   ", "  ")
		fmt.Printf("   Args:\n%s\n", string(argsJSON))
	}

	// Execute the tool via registry
	resp, err := s.registry.Execute(ctx, funcCall)
	if err != nil {
		fmt.Printf("❌ Tool execution failed: %v\n", err)
		return nil, goerr.Wrap(err, "tool execution failed")
	}

	// Check if response contains error
	if errMsg, ok := resp.Response["error"].(string); ok {
		fmt.Printf("⚠️  Tool returned error:\n%s\n\n", errMsg)
		return resp, nil
	}

	// Display result preview (first 200 chars)
	if result, ok := resp.Response["result"].(string); ok {
		resultPreview := result
		if len(result) > 200 {
			resultPreview = result[:200] + "..."
		}
		fmt.Printf("✅ Tool result:\n%s\n\n", resultPreview)
	} else {
		// Display other response fields
		respJSON, _ := json.MarshalIndent(resp.Response, "", "  ")
		resultPreview := string(respJSON)
		if len(resultPreview) > 200 {
			resultPreview = resultPreview[:200] + "..."
		}
		fmt.Printf("✅ Tool result:\n%s\n\n", resultPreview)
	}

	return resp, nil
}
