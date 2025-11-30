package chat

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

	alertID         model.AlertID
	alert           *model.Alert
	history         *model.History
	environmentInfo string
}

//go:embed prompt/session.md
var sessionPromptRaw string

var sessionPromptTmpl = template.Must(template.New("session").Parse(sessionPromptRaw))

// NewInput contains parameters for creating a new chat session
type NewInput struct {
	Repo            repository.Repository
	Gemini          adapter.Gemini
	Storage         adapter.Storage
	Registry        *tool.Registry
	AlertID         model.AlertID
	HistoryID       *model.HistoryID // Optional: specify to continue existing conversation
	EnvironmentInfo string           // Optional: environment context for better analysis
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

		alertID:         input.AlertID,
		alert:           alert,
		history:         history,
		environmentInfo: input.EnvironmentInfo,
	}, nil
}

func (s *Session) Send(ctx context.Context, message string) (*genai.GenerateContentResponse, error) {
	// Check if plan & execute mode should be used
	if shouldUsePlanExecuteMode(ctx, s.gemini, message, s.history.Contents) {
		// Use plan & execute mode
		result, err := s.SendWithPlanExecute(ctx, message)
		if err != nil {
			return nil, goerr.Wrap(err, "Plan & Execute mode failed")
		}
		// Plan & Execute mode succeeded
		// Convert to response format (create a synthetic response)
		return s.createResponseFromPlanExecute(result), nil
	}

	// Direct mode (existing logic)
	// Generate title from first user input if this is a new history
	if len(s.history.Contents) == 0 {
		title, err := generateTitle(ctx, s.gemini, message)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to generate title")
		}
		s.history.Title = title
	}

	// Build system prompt using template
	systemPrompt, err := s.buildSystemPrompt(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build system prompt")
	}

	// Add user message to history
	userContent := genai.NewContentFromText(message, genai.RoleUser)
	s.history.Contents = append(s.history.Contents, userContent)

	// Build config with system instruction and tools
	thinkingBudget := int32(0)
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, ""),
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: false,
			ThinkingBudget:  &thinkingBudget,
		},
	}

	// Add tools from registry if available
	if s.registry != nil {
		config.Tools = s.registry.Specs()
	}

	// Tool Call loop: keep generating until no more function calls
	const maxIterations = 32
	var finalResp *genai.GenerateContentResponse

	for i := 0; i < maxIterations; i++ {
		resp, err := s.gemini.GenerateContent(ctx, s.history.Contents, config)
		if err != nil {
			// Check if error is due to token limit exceeded
			if isTokenLimitError(err) {
				// Attempt compression
				fmt.Println("\n📦 Token limit exceeded. Compressing conversation history...")

				compressedContents, compressErr := compressHistory(ctx, s.gemini, s.history.Contents)
				if compressErr != nil {
					return nil, goerr.Wrap(compressErr, "failed to compress history")
				}

				// Update history with compressed contents
				s.history.Contents = compressedContents

				// Save compressed history immediately
				if saveErr := saveHistory(ctx, s.repo, s.storage, s.alertID, s.history); saveErr != nil {
					fmt.Printf("⚠️  警告: 圧縮した履歴の保存に失敗しました: %v\n", saveErr)
				}

				fmt.Println("✅ 会話履歴を圧縮しました。リトライします...")
				continue // Retry with compressed history
			}
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
		var functionResponses []*genai.Part
		for _, candidate := range resp.Candidates {
			if candidate.Content == nil {
				continue
			}

			// Add function call content to history
			s.history.Contents = append(s.history.Contents, candidate.Content)

			for _, part := range candidate.Content.Parts {
				// Display text content if present
				if part.Text != "" {
					fmt.Printf("\n💭 %s\n\n", part.Text)
				}

				if part.FunctionCall == nil {
					continue
				}

				// Execute the tool
				funcResp, err := s.executeTool(ctx, *part.FunctionCall)
				if err != nil {
					// Display error to user
					fmt.Printf("❌ ツールエラー (%s): %v\n", part.FunctionCall.Name, err)
					// Create error response
					funcResp = &genai.FunctionResponse{
						Name:     part.FunctionCall.Name,
						Response: map[string]any{"error": err.Error()},
					}
				}

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
			s.history.Contents = append(s.history.Contents, funcRespContent)
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
	fmt.Printf("\n🔧 ツール呼び出し: %s\n", funcCall.Name)
	if funcCall.Args != nil {
		argsJSON, _ := json.MarshalIndent(funcCall.Args, "   ", "  ")
		fmt.Printf("   引数:\n%s\n", string(argsJSON))
	}

	// Execute the tool via registry
	resp, err := s.registry.Execute(ctx, funcCall)
	if err != nil {
		fmt.Printf("❌ ツール実行失敗: %v\n", err)
		return nil, goerr.Wrap(err, "tool execution failed")
	}

	// Check if response contains error
	if errMsg, ok := resp.Response["error"].(string); ok {
		fmt.Printf("⚠️  ツールエラー:\n%s\n\n", errMsg)
		return resp, nil
	}

	// Display result preview (first 200 chars)
	if result, ok := resp.Response["result"].(string); ok {
		resultPreview := result
		if len(result) > 200 {
			resultPreview = result[:200] + "..."
		}
		fmt.Printf("✅ ツール実行結果:\n%s\n\n", resultPreview)
	} else {
		// Display other response fields
		respJSON, _ := json.MarshalIndent(resp.Response, "", "  ")
		resultPreview := string(respJSON)
		if len(resultPreview) > 200 {
			resultPreview = resultPreview[:200] + "..."
		}
		fmt.Printf("✅ ツール実行結果:\n%s\n\n", resultPreview)
	}

	return resp, nil
}

func (s *Session) buildSystemPrompt(ctx context.Context) (string, error) {
	// Marshal alert data
	alertData, err := json.MarshalIndent(s.alert.Data, "", "  ")
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal alert data")
	}

	// Collect tool-specific prompts from registry
	toolPrompts := ""
	if s.registry != nil {
		toolPrompts = s.registry.Prompts(ctx)
	}

	// Execute template
	var buf bytes.Buffer
	if err := sessionPromptTmpl.Execute(&buf, map[string]any{
		"AlertID":         s.alertID,
		"Alert":           s.alert,
		"AlertData":       string(alertData),
		"EnvironmentInfo": s.environmentInfo,
		"ToolPrompts":     toolPrompts,
	}); err != nil {
		return "", goerr.Wrap(err, "failed to execute session prompt template")
	}

	return buf.String(), nil
}

// SendWithPlanExecute executes the plan & execute mode
func (s *Session) SendWithPlanExecute(ctx context.Context, message string) (*PlanExecuteResult, error) {
	// Generate title from request if this is a new history
	if len(s.history.Contents) == 0 {
		title, err := generateTitle(ctx, s.gemini, message)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to generate title")
		}
		s.history.Title = title
	}

	// Initialize plan & execute components
	planGen := newPlanGenerator(s.gemini, s.registry)
	conclusionGen := newConclusionGenerator(s.gemini)

	// Step 1: Generate plan
	fmt.Printf("\n📋 計画を生成中...\n")
	plan, err := planGen.Generate(ctx, message, s.alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate plan")
	}
	displayPlan(plan)

	// Step 2-4: Execute plan with reflection loop
	results, reflections, err := executeStepsWithReflection(ctx, s.gemini, s.registry, plan)
	if err != nil {
		return nil, err
	}

	// Step 5: Generate conclusion
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📝 結論を生成中...\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	conclusion, err := conclusionGen.Generate(ctx, plan, results, reflections)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate conclusion")
	}

	// Add to history
	// User message
	userContent := &genai.Content{
		Role:  genai.RoleUser,
		Parts: []*genai.Part{{Text: message}},
	}
	s.history.Contents = append(s.history.Contents, userContent)

	// Assistant response (conclusion)
	assistantText := fmt.Sprintf("## 完了\n\n**目的**: %s\n\n%s", plan.Objective, conclusion.Content)

	assistantContent := &genai.Content{
		Role:  genai.RoleModel,
		Parts: []*genai.Part{{Text: assistantText}},
	}
	s.history.Contents = append(s.history.Contents, assistantContent)

	// Save to history
	if err := saveHistory(ctx, s.repo, s.storage, s.alertID, s.history); err != nil {
		fmt.Printf("⚠️  警告: 履歴の保存に失敗しました: %v\n", err)
	}

	return &PlanExecuteResult{
		Plan:        plan,
		Results:     results,
		Reflections: reflections,
		Conclusion:  conclusion,
	}, nil
}

// createResponseFromPlanExecute creates a synthetic response from plan & execute result
func (s *Session) createResponseFromPlanExecute(result *PlanExecuteResult) *genai.GenerateContentResponse {
	// Format conclusion as text
	var text bytes.Buffer
	text.WriteString("## 完了\n\n")
	text.WriteString(fmt.Sprintf("**目的**: %s\n\n", result.Plan.Objective))
	text.WriteString(result.Conclusion.Content)

	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Role: genai.RoleModel,
					Parts: []*genai.Part{
						{Text: text.String()},
					},
				},
			},
		},
	}
}
