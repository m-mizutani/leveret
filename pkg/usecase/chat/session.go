package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
	"github.com/m-mizutani/leveret/pkg/tool/alert"
	"google.golang.org/genai"
)

// Session manages an interactive chat session for alert analysis
type Session struct {
	repo        repository.Repository
	gemini      adapter.Gemini
	storage     adapter.Storage
	searchAlert *alert.SearchAlerts

	alertID model.AlertID
	alert   *model.Alert
	history *model.History
}

// NewInput contains parameters for creating a new chat session
type NewInput struct {
	Repo        repository.Repository
	Gemini      adapter.Gemini
	Storage     adapter.Storage
	SearchAlert *alert.SearchAlerts
	AlertID     model.AlertID
	HistoryID   *model.HistoryID // Optional: specify to continue existing conversation
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
		repo:        input.Repo,
		gemini:      input.Gemini,
		storage:     input.Storage,
		searchAlert: input.SearchAlert,

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

	// Add user message to history
	userContent := genai.NewContentFromText(message, genai.RoleUser)
	s.history.Contents = append(s.history.Contents, userContent)

	// Build config with system instruction and tools
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, ""),
	}

	// Add tool if available
	if s.searchAlert != nil {
		config.Tools = []*genai.Tool{{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				s.searchAlert.FunctionDeclaration(),
			},
		}}
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
				funcCall := part.FunctionCall
				if funcCall == nil {
					continue
				}

				// Execute the tool
				result, err := s.executeTool(ctx, funcCall)
				if err != nil {
					result = "Error: " + err.Error()
				}

				// Add function response to history
				funcResp := &genai.FunctionResponse{
					Name:     funcCall.Name,
					Response: map[string]any{"result": result},
				}
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

func (s *Session) executeTool(ctx context.Context, funcCall *genai.FunctionCall) (string, error) {
	if s.searchAlert == nil {
		return "", goerr.New("tool not available")
	}

	// Marshal function call arguments to JSON
	paramsJSON, err := json.Marshal(funcCall.Args)
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal function arguments")
	}

	// Display tool call information
	fmt.Printf("\n🔧 Calling tool: %s\n", funcCall.Name)
	var argsFormatted map[string]any
	if err := json.Unmarshal(paramsJSON, &argsFormatted); err == nil {
		for key, value := range argsFormatted {
			fmt.Printf("   %s: %v\n", key, value)
		}
	}

	// Execute the tool
	result, err := s.searchAlert.Execute(ctx, paramsJSON)
	if err != nil {
		fmt.Printf("❌ Tool execution failed: %v\n", err)
		return "", goerr.Wrap(err, "tool execution failed")
	}

	// Display result preview (first 200 chars)
	resultPreview := result
	if len(result) > 200 {
		resultPreview = result[:200] + "..."
	}
	fmt.Printf("✅ Tool result:\n%s\n\n", resultPreview)

	return result, nil
}
