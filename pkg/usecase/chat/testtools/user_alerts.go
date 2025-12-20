package testtools

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

// listUsersTool returns a list of test users
type listUsersTool struct {
	callCount int
}

func NewListUsersTool() *listUsersTool {
	return &listUsersTool{callCount: 0}
}

func (t *listUsersTool) CallCount() int {
	return t.callCount
}

func (t *listUsersTool) Flags() []cli.Flag {
	return nil
}

func (t *listUsersTool) Init(ctx context.Context, client *tool.Client) (bool, error) {
	return true, nil
}

func (t *listUsersTool) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "list_users",
				Description: "Get a list of all users in the system",
				Parameters: &genai.Schema{
					Type:       genai.TypeObject,
					Properties: map[string]*genai.Schema{},
					Required:   []string{},
				},
			},
		},
	}
}

func (t *listUsersTool) Prompt(ctx context.Context) string {
	return ""
}

func (t *listUsersTool) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	t.callCount++
	users := []string{"user_a", "user_b", "user_c"}

	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"users": users,
		},
	}, nil
}

// getUserAlertsTool returns alert statistics for a specific user
type getUserAlertsTool struct {
	callCount int
	userIDs   []string
}

func NewGetUserAlertsTool() *getUserAlertsTool {
	return &getUserAlertsTool{callCount: 0, userIDs: []string{}}
}

func (t *getUserAlertsTool) CallCount() int {
	return t.callCount
}

func (t *getUserAlertsTool) CalledUserIDs() []string {
	return t.userIDs
}

func (t *getUserAlertsTool) Flags() []cli.Flag {
	return nil
}

func (t *getUserAlertsTool) Init(ctx context.Context, client *tool.Client) (bool, error) {
	return true, nil
}

func (t *getUserAlertsTool) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_user_alerts",
				Description: "Get alert statistics for a specific user",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"user_id": {
							Type:        genai.TypeString,
							Description: "User ID to get alerts for",
						},
					},
					Required: []string{"user_id"},
				},
			},
		},
	}
}

func (t *getUserAlertsTool) Prompt(ctx context.Context) string {
	return ""
}

func (t *getUserAlertsTool) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	t.callCount++

	userID, ok := fc.Args["user_id"].(string)
	if !ok {
		return nil, goerr.New("user_id argument is required")
	}
	t.userIDs = append(t.userIDs, userID)

	// Test data with predetermined risk scores
	// user_a: 3 * 5.0 = 15.0
	// user_b: 5 * 4.0 = 20.0 (highest)
	// user_c: 2 * 7.0 = 14.0
	var count, severitySum int
	switch userID {
	case "user_a":
		count = 3
		severitySum = 15
	case "user_b":
		count = 5
		severitySum = 20
	case "user_c":
		count = 2
		severitySum = 14
	default:
		return nil, goerr.New("unknown user_id", goerr.V("user_id", userID))
	}

	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"count":        count,
			"severity_sum": severitySum,
		},
	}, nil
}
