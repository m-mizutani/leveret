package chat_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/m-mizutani/leveret/pkg/usecase/chat"
	"github.com/m-mizutani/leveret/pkg/usecase/chat/testtools"
	"google.golang.org/genai"
)

// Mock Repository
type mockRepository struct {
	alerts    map[model.AlertID]*model.Alert
	histories map[model.HistoryID]*model.History
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		alerts:    make(map[model.AlertID]*model.Alert),
		histories: make(map[model.HistoryID]*model.History),
	}
}

func (m *mockRepository) GetAlert(ctx context.Context, id model.AlertID) (*model.Alert, error) {
	alert, ok := m.alerts[id]
	if !ok {
		return nil, goerr.New("alert not found", goerr.V("alert_id", id))
	}
	return alert, nil
}

func (m *mockRepository) GetHistory(ctx context.Context, id model.HistoryID) (*model.History, error) {
	history, ok := m.histories[id]
	if !ok {
		return nil, goerr.New("history not found", goerr.V("history_id", id))
	}
	return history, nil
}

// Implement other required methods as no-op or minimal implementations
func (m *mockRepository) PutAlert(ctx context.Context, alert *model.Alert) error {
	m.alerts[alert.ID] = alert
	return nil
}

func (m *mockRepository) ListAlerts(ctx context.Context, offset, limit int) ([]*model.Alert, error) {
	return nil, nil
}

func (m *mockRepository) SearchAlerts(ctx context.Context, input *repository.SearchAlertsInput) ([]*model.Alert, error) {
	return nil, nil
}

func (m *mockRepository) SearchSimilarAlerts(ctx context.Context, embedding []float32, threshold float64) ([]*model.Alert, error) {
	return nil, nil
}

func (m *mockRepository) PutHistory(ctx context.Context, history *model.History) error {
	if history.ID == "" {
		history.ID = model.NewHistoryID()
	}
	m.histories[history.ID] = history
	return nil
}

func (m *mockRepository) ListHistory(ctx context.Context, offset, limit int) ([]*model.History, error) {
	return nil, nil
}

func (m *mockRepository) ListHistoryByAlert(ctx context.Context, alertID model.AlertID) ([]*model.History, error) {
	return nil, nil
}

func (m *mockRepository) PutMemory(ctx context.Context, memory *model.Memory) error {
	return nil
}

func (m *mockRepository) GetMemory(ctx context.Context, id model.MemoryID) (*model.Memory, error) {
	return nil, nil
}

func (m *mockRepository) SearchMemories(ctx context.Context, embedding firestore.Vector32, threshold float64, limit int) ([]*model.Memory, error) {
	return nil, nil
}

func (m *mockRepository) UpdateMemoryScore(ctx context.Context, id model.MemoryID, delta float64) error {
	return nil
}

func (m *mockRepository) DeleteMemoriesBelowScore(ctx context.Context, threshold float64) error {
	return nil
}

// Mock Storage
type mockStorage struct {
	data map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: make(map[string][]byte),
	}
}

func (m *mockStorage) Put(ctx context.Context, key string) (io.WriteCloser, error) {
	buf := &bytes.Buffer{}
	return &mockWriteCloser{
		Buffer:  buf,
		storage: m,
		key:     key,
	}, nil
}

type mockWriteCloser struct {
	*bytes.Buffer
	storage *mockStorage
	key     string
}

func (m *mockWriteCloser) Close() error {
	m.storage.data[m.key] = m.Buffer.Bytes()
	return nil
}

func (m *mockStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	data, ok := m.data[key]
	if !ok {
		return nil, goerr.New("data not found", goerr.V("key", key))
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Helper functions

// answerValidationRequest is the request for LLM answer validation
type answerValidationRequest struct {
	Question       string
	ExpectedAnswer string
	ActualAnswer   string
}

// answerValidationResponse is the response from LLM answer validation
type answerValidationResponse struct {
	IsValid     bool   `json:"is_valid"`
	Explanation string `json:"explanation"`
}

// validateAnswer validates if the actual answer contains the expected information using LLM
func validateAnswer(ctx context.Context, gemini adapter.Gemini, req answerValidationRequest) (*answerValidationResponse, error) {
	prompt := `You are validating if an AI assistant's answer contains the expected information.

Original question: ` + req.Question + `
Expected answer should contain: ` + req.ExpectedAnswer + `
Actual answer: ` + req.ActualAnswer + `

Respond in JSON format with:
- is_valid (boolean): true if the actual answer contains the expected information
- explanation (string): brief explanation of your judgment`

	config := &genai.GenerateContentConfig{
		Temperature: ptrFloat32(0.0),
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"is_valid": {
					Type:        genai.TypeBoolean,
					Description: "true if the actual answer contains the expected information",
				},
				"explanation": {
					Type:        genai.TypeString,
					Description: "brief explanation of your judgment",
				},
			},
			Required: []string{"is_valid", "explanation"},
		},
	}

	contents := []*genai.Content{
		genai.NewContentFromText(prompt, genai.RoleUser),
	}

	resp, err := gemini.GenerateContent(ctx, contents, config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate validation")
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, goerr.New("no validation response")
	}

	text := resp.Candidates[0].Content.Parts[0].Text
	var validation answerValidationResponse
	if err := json.Unmarshal([]byte(text), &validation); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal validation response")
	}

	return &validation, nil
}

// extractResponseText extracts text from response
func extractResponseText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return ""
	}
	var parts []string
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// testSessionHelper wraps session components for testing
type testSessionHelper struct {
	Session *chat.Session
	Gemini  adapter.Gemini
	Repo    *mockRepository
	Storage *mockStorage
	AlertID model.AlertID
}

// setupTestSession creates a test session with real Gemini and mock dependencies
func setupTestSession(ctx context.Context, t *testing.T, tools []tool.Tool) *testSessionHelper {
	// Check environment variable
	projectID := os.Getenv("TEST_GEMINI_PROJECT")
	if projectID == "" {
		t.Skip("TEST_GEMINI_PROJECT not set, skipping integration test")
	}

	// Create real Gemini client
	gemini, err := adapter.NewGemini(ctx, projectID, "us-central1")
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}

	// Create mock repository and storage
	repo := newMockRepository()
	storage := newMockStorage()

	// Create test alert
	alertID := model.NewAlertID()
	alert := &model.Alert{
		ID:          alertID,
		Title:       "Test Alert",
		Description: "This is a test alert for integration testing",
		Data:        map[string]any{"test": "data"},
		CreatedAt:   time.Now(),
	}
	repo.alerts[alertID] = alert

	// Create tool registry
	registry := tool.New(tools...)
	client := &tool.Client{}
	if err := registry.Init(ctx, client); err != nil {
		t.Fatalf("failed to initialize tool registry: %v", err)
	}

	// Create session
	session, err := chat.New(ctx, chat.NewInput{
		Repo:     repo,
		Gemini:   gemini,
		Storage:  storage,
		Registry: registry,
		AlertID:  alertID,
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	return &testSessionHelper{
		Session: session,
		Gemini:  gemini,
		Repo:    repo,
		Storage: storage,
		AlertID: alertID,
	}
}

func ptrFloat32(f float32) *float32 {
	return &f
}

// Test: Scenario 1 - User Risk Analysis
func TestChatSession_UserRiskAnalysis(t *testing.T) {
	ctx := context.Background()

	listUsers := testtools.NewListUsersTool()
	getUserAlerts := testtools.NewGetUserAlertsTool()

	helper := setupTestSession(ctx, t, []tool.Tool{listUsers, getUserAlerts})

	// Send task
	task := "Find out which user has the most critical alerts. Get all users, then for each user get their alert count and severity sum, and determine who has the highest risk score (count * average_severity)."
	resp, err := helper.Session.Send(ctx, task)
	gt.NoError(t, err)

	// Extract response text
	answerText := extractResponseText(resp)

	// Validate tool calls
	gt.Number(t, listUsers.CallCount()).GreaterOrEqual(1).Describe("list_users should be called at least once")
	gt.Number(t, getUserAlerts.CallCount()).GreaterOrEqual(1).Describe("get_user_alerts should be called at least once")

	// Validate tool arguments - getUserAlerts should be called for all users
	calledUsers := getUserAlerts.CalledUserIDs()
	userSet := make(map[string]bool)
	for _, u := range calledUsers {
		userSet[u] = true
	}
	gt.True(t, userSet["user_a"] && userSet["user_b"] && userSet["user_c"]).Describe("get_user_alerts should be called for all users (user_a, user_b, user_c)")

	// Validate answer using LLM
	validation, err := validateAnswer(ctx, helper.Gemini, answerValidationRequest{
		Question:       task,
		ExpectedAnswer: "user_b has the highest risk score",
		ActualAnswer:   answerText,
	})
	gt.NoError(t, err)
	gt.True(t, validation.IsValid).Describe("Answer validation failed: " + validation.Explanation)
}

// Test: Scenario 2 - Large-scale Log Analysis / Unauthorized Login Detection
func TestChatSession_UnauthorizedLoginDetection(t *testing.T) {
	ctx := context.Background()

	getAuthLogs := testtools.NewGetAuthLogsTool()
	getUserInfo := testtools.NewGetUserInfoTool()

	helper := setupTestSession(ctx, t, []tool.Tool{getAuthLogs, getUserInfo})

	// Send task
	task := "Analyze authentication logs from multiple sources to detect unauthorized login attempts. Get logs from web server, VPN server, and database server for the last hour, then lookup user information for suspicious patterns, and identify potential security breaches."
	resp, err := helper.Session.Send(ctx, task)
	gt.NoError(t, err)

	// Extract response text
	answerText := extractResponseText(resp)

	// Validate tool calls
	gt.Number(t, getAuthLogs.CallCount()).GreaterOrEqual(1).Describe("get_auth_logs should be called at least once")
	gt.Number(t, getUserInfo.CallCount()).GreaterOrEqual(1).Describe("get_user_info should be called at least once")

	// Validate tool arguments - getAuthLogs should be called for all 3 sources
	calledSources := getAuthLogs.CalledSources()
	sourceSet := make(map[string]bool)
	for _, s := range calledSources {
		sourceSet[s] = true
	}
	gt.True(t, sourceSet["web_server"] && sourceSet["vpn_server"] && sourceSet["database_server"]).Describe("get_auth_logs should be called for all sources (web_server, vpn_server, database_server)")

	// Validate tool arguments - getUserInfo should be called for bob
	calledUserIDs := getUserInfo.CalledUserIDs()
	foundBob := false
	for _, uid := range calledUserIDs {
		if uid == "bob" {
			foundBob = true
			break
		}
	}
	gt.True(t, foundBob).Describe("get_user_info should be called with user_id 'bob'")

	// Validate answer using LLM
	validation, err := validateAnswer(ctx, helper.Gemini, answerValidationRequest{
		Question:       task,
		ExpectedAnswer: "bob account shows unauthorized login attempts or suspicious activity",
		ActualAnswer:   answerText,
	})
	gt.NoError(t, err)
	gt.True(t, validation.IsValid).Describe("Answer validation failed: " + validation.Explanation)
}

// Test: Scenario 3 - Security Incident Analysis
func TestChatSession_SecurityIncidentAnalysis(t *testing.T) {
	ctx := context.Background()

	getAlertDetails := testtools.NewGetAlertDetailsTool()
	checkIPReputation := testtools.NewCheckIPReputationTool()
	getHistoricalAlerts := testtools.NewGetHistoricalAlertsTool()

	helper := setupTestSession(ctx, t, []tool.Tool{getAlertDetails, checkIPReputation, getHistoricalAlerts})

	// Send task (use the actual alert ID from the session)
	task := fmt.Sprintf("For alert_id '%s', determine if it's a true security incident. You MUST: 1) Get alert details to find source IP, 2) Check the source IP reputation, 3) Get historical alerts from the same source IP, and 4) Analyze the attack pattern based on all this information to make a final judgment on whether this is a true positive.", helper.AlertID)
	resp, err := helper.Session.Send(ctx, task)
	gt.NoError(t, err)

	// Extract response text
	answerText := extractResponseText(resp)

	// Validate tool calls
	gt.Number(t, getAlertDetails.CallCount()).GreaterOrEqual(1).Describe("get_alert_details should be called at least once")
	gt.Number(t, checkIPReputation.CallCount()).GreaterOrEqual(1).Describe("check_ip_reputation should be called at least once")
	gt.Number(t, getHistoricalAlerts.CallCount()).GreaterOrEqual(1).Describe("get_historical_alerts should be called at least once")

	// Validate tool arguments
	calledAlertIDs := getAlertDetails.CalledAlertIDs()
	foundAlertID := false
	for _, aid := range calledAlertIDs {
		if aid == string(helper.AlertID) {
			foundAlertID = true
			break
		}
	}
	gt.True(t, foundAlertID).Describe("get_alert_details should be called with the correct alert_id")

	calledIPs := checkIPReputation.CalledIPs()
	foundIP := false
	for _, ip := range calledIPs {
		if ip == "203.0.113.42" {
			foundIP = true
			break
		}
	}
	gt.True(t, foundIP).Describe("check_ip_reputation should be called with IP '203.0.113.42'")

	calledSourceIPs := getHistoricalAlerts.CalledSourceIPs()
	foundSourceIP := false
	for _, ip := range calledSourceIPs {
		if ip == "203.0.113.42" {
			foundSourceIP = true
			break
		}
	}
	gt.True(t, foundSourceIP).Describe("get_historical_alerts should be called with source_ip '203.0.113.42'")

	// Validate answer using LLM
	validation, err := validateAnswer(ctx, helper.Gemini, answerValidationRequest{
		Question:       task,
		ExpectedAnswer: fmt.Sprintf("alert %s is a true security incident (true positive)", helper.AlertID),
		ActualAnswer:   answerText,
	})
	gt.NoError(t, err)
	gt.True(t, validation.IsValid).Describe("Answer validation failed: " + validation.Explanation)
}
