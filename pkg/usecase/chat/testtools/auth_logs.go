package testtools

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

// getAuthLogsTool returns authentication logs from various sources
type getAuthLogsTool struct {
	callCount int
	sources   []string
}

func NewGetAuthLogsTool() *getAuthLogsTool {
	return &getAuthLogsTool{callCount: 0, sources: []string{}}
}

func (t *getAuthLogsTool) CallCount() int {
	return t.callCount
}

func (t *getAuthLogsTool) CalledSources() []string {
	return t.sources
}

func (t *getAuthLogsTool) Flags() []cli.Flag {
	return nil
}

func (t *getAuthLogsTool) Init(ctx context.Context, client *tool.Client) (bool, error) {
	return true, nil
}

func (t *getAuthLogsTool) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_auth_logs",
				Description: "Get authentication logs from a specific source (web_server, vpn_server, or database_server)",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"source": {
							Type:        genai.TypeString,
							Description: "Log source: web_server, vpn_server, or database_server",
						},
					},
					Required: []string{"source"},
				},
			},
		},
	}
}

func (t *getAuthLogsTool) Prompt(ctx context.Context) string {
	return ""
}

func (t *getAuthLogsTool) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	t.callCount++

	source, ok := fc.Args["source"].(string)
	if !ok {
		return nil, goerr.New("source argument is required")
	}
	t.sources = append(t.sources, source)

	var logs []map[string]any

	switch source {
	case "web_server":
		logs = []map[string]any{
			{"log_id": "web_001", "timestamp": "2024-01-01T14:00:15Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "web_002", "timestamp": "2024-01-01T14:05:22Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "web_003", "timestamp": "2024-01-01T14:10:33Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "web_004", "timestamp": "2024-01-01T14:12:45Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "web_005", "timestamp": "2024-01-01T14:15:01Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "web_006", "timestamp": "2024-01-01T14:18:22Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "web_007", "timestamp": "2024-01-01T14:20:33Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "web_008", "timestamp": "2024-01-01T14:25:44Z", "user_id": "david", "source_ip": "192.168.1.130", "status": "success", "location": "Osaka"},
			{"log_id": "web_009", "timestamp": "2024-01-01T14:28:55Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "web_010", "timestamp": "2024-01-01T14:30:11Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "web_011", "timestamp": "2024-01-01T14:32:22Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "web_012", "timestamp": "2024-01-01T14:35:33Z", "user_id": "eve", "source_ip": "192.168.1.140", "status": "success", "location": "Tokyo"},
			{"log_id": "web_013", "timestamp": "2024-01-01T14:38:44Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "web_014", "timestamp": "2024-01-01T14:40:55Z", "user_id": "david", "source_ip": "192.168.1.130", "status": "success", "location": "Osaka"},
			{"log_id": "web_015", "timestamp": "2024-01-01T14:42:11Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "web_016", "timestamp": "2024-01-01T14:45:22Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "web_017", "timestamp": "2024-01-01T14:48:33Z", "user_id": "eve", "source_ip": "192.168.1.140", "status": "success", "location": "Tokyo"},
			{"log_id": "web_018", "timestamp": "2024-01-01T14:50:44Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "web_019", "timestamp": "2024-01-01T14:52:55Z", "user_id": "david", "source_ip": "192.168.1.130", "status": "success", "location": "Osaka"},
			{"log_id": "web_020", "timestamp": "2024-01-01T14:55:11Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
		}
	case "vpn_server":
		logs = []map[string]any{
			{"log_id": "vpn_001", "timestamp": "2024-01-01T14:01:20Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "vpn_002", "timestamp": "2024-01-01T14:06:30Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "vpn_003", "timestamp": "2024-01-01T14:11:40Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "vpn_004", "timestamp": "2024-01-01T14:16:50Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "vpn_005", "timestamp": "2024-01-01T14:21:00Z", "user_id": "david", "source_ip": "192.168.1.130", "status": "success", "location": "Osaka"},
			{"log_id": "vpn_006", "timestamp": "2024-01-01T14:26:10Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "vpn_007", "timestamp": "2024-01-01T14:31:20Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "vpn_008", "timestamp": "2024-01-01T14:36:30Z", "user_id": "eve", "source_ip": "192.168.1.140", "status": "success", "location": "Tokyo"},
			{"log_id": "vpn_009", "timestamp": "2024-01-01T14:41:40Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "vpn_010", "timestamp": "2024-01-01T14:46:50Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "vpn_011", "timestamp": "2024-01-01T14:51:00Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "vpn_012", "timestamp": "2024-01-01T14:56:10Z", "user_id": "david", "source_ip": "192.168.1.130", "status": "success", "location": "Osaka"},
			{"log_id": "vpn_013", "timestamp": "2024-01-01T15:01:20Z", "user_id": "eve", "source_ip": "192.168.1.140", "status": "success", "location": "Tokyo"},
			{"log_id": "vpn_014", "timestamp": "2024-01-01T15:06:30Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "vpn_015", "timestamp": "2024-01-01T15:11:40Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
		}
	case "database_server":
		logs = []map[string]any{
			{"log_id": "db_001", "timestamp": "2024-01-01T14:02:25Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "db_002", "timestamp": "2024-01-01T14:07:35Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "db_003", "timestamp": "2024-01-01T14:12:45Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "db_004", "timestamp": "2024-01-01T14:17:55Z", "user_id": "david", "source_ip": "192.168.1.130", "status": "success", "location": "Osaka"},
			{"log_id": "db_005", "timestamp": "2024-01-01T14:23:05Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "db_006", "timestamp": "2024-01-01T14:28:15Z", "user_id": "eve", "source_ip": "192.168.1.140", "status": "success", "location": "Tokyo"},
			{"log_id": "db_007", "timestamp": "2024-01-01T14:33:25Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "db_008", "timestamp": "2024-01-01T14:38:35Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "db_009", "timestamp": "2024-01-01T14:43:45Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "db_010", "timestamp": "2024-01-01T14:48:55Z", "user_id": "david", "source_ip": "192.168.1.130", "status": "success", "location": "Osaka"},
			{"log_id": "db_011", "timestamp": "2024-01-01T14:54:05Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "db_012", "timestamp": "2024-01-01T14:59:15Z", "user_id": "eve", "source_ip": "192.168.1.140", "status": "success", "location": "Tokyo"},
			{"log_id": "db_013", "timestamp": "2024-01-01T15:04:25Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "db_014", "timestamp": "2024-01-01T15:09:35Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "db_015", "timestamp": "2024-01-01T15:14:45Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "db_016", "timestamp": "2024-01-01T15:19:55Z", "user_id": "david", "source_ip": "192.168.1.130", "status": "success", "location": "Osaka"},
			{"log_id": "db_017", "timestamp": "2024-01-01T15:25:05Z", "user_id": "eve", "source_ip": "192.168.1.140", "status": "success", "location": "Tokyo"},
			{"log_id": "db_018", "timestamp": "2024-01-01T15:30:15Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "db_019", "timestamp": "2024-01-01T15:35:25Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "db_020", "timestamp": "2024-01-01T15:40:35Z", "user_id": "david", "source_ip": "192.168.1.130", "status": "success", "location": "Osaka"},
			{"log_id": "db_021", "timestamp": "2024-01-01T15:45:45Z", "user_id": "bob", "source_ip": "203.0.113.50", "status": "failed", "location": "Unknown"},
			{"log_id": "db_022", "timestamp": "2024-01-01T15:50:55Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
			{"log_id": "db_023", "timestamp": "2024-01-01T15:56:05Z", "user_id": "eve", "source_ip": "192.168.1.140", "status": "success", "location": "Tokyo"},
			{"log_id": "db_024", "timestamp": "2024-01-01T16:01:15Z", "user_id": "charlie", "source_ip": "192.168.1.120", "status": "success", "location": "Tokyo"},
			{"log_id": "db_025", "timestamp": "2024-01-01T16:06:25Z", "user_id": "alice", "source_ip": "192.168.1.100", "status": "success", "location": "Tokyo"},
		}
	default:
		return nil, goerr.New("unknown source", goerr.V("source", source))
	}

	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"source": source,
			"logs":   logs,
			"count":  len(logs),
		},
	}, nil
}

// getUserInfoTool returns user information including normal access patterns
type getUserInfoTool struct {
	callCount int
	userIDs   []string
}

func NewGetUserInfoTool() *getUserInfoTool {
	return &getUserInfoTool{callCount: 0, userIDs: []string{}}
}

func (t *getUserInfoTool) CallCount() int {
	return t.callCount
}

func (t *getUserInfoTool) CalledUserIDs() []string {
	return t.userIDs
}

func (t *getUserInfoTool) Flags() []cli.Flag {
	return nil
}

func (t *getUserInfoTool) Init(ctx context.Context, client *tool.Client) (bool, error) {
	return true, nil
}

func (t *getUserInfoTool) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_user_info",
				Description: "Get user information including normal access patterns",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"user_id": {
							Type:        genai.TypeString,
							Description: "User ID to get information for",
						},
					},
					Required: []string{"user_id"},
				},
			},
		},
	}
}

func (t *getUserInfoTool) Prompt(ctx context.Context) string {
	return ""
}

func (t *getUserInfoTool) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	t.callCount++

	userID, ok := fc.Args["user_id"].(string)
	if !ok {
		return nil, goerr.New("user_id argument is required")
	}
	t.userIDs = append(t.userIDs, userID)

	var userInfo map[string]any
	switch userID {
	case "alice":
		userInfo = map[string]any{
			"user_id":          "alice",
			"department":       "Engineering",
			"normal_locations": []string{"Tokyo"},
			"last_known_ip":    "192.168.1.100",
			"account_status":   "active",
		}
	case "bob":
		userInfo = map[string]any{
			"user_id":          "bob",
			"department":       "Engineering",
			"normal_locations": []string{"Tokyo", "Osaka"},
			"last_known_ip":    "192.168.1.150",
			"account_status":   "active",
		}
	case "charlie":
		userInfo = map[string]any{
			"user_id":          "charlie",
			"department":       "Sales",
			"normal_locations": []string{"Tokyo"},
			"last_known_ip":    "192.168.1.120",
			"account_status":   "active",
		}
	case "david":
		userInfo = map[string]any{
			"user_id":          "david",
			"department":       "HR",
			"normal_locations": []string{"Osaka"},
			"last_known_ip":    "192.168.1.130",
			"account_status":   "active",
		}
	case "eve":
		userInfo = map[string]any{
			"user_id":          "eve",
			"department":       "Marketing",
			"normal_locations": []string{"Tokyo"},
			"last_known_ip":    "192.168.1.140",
			"account_status":   "active",
		}
	default:
		return nil, goerr.New("unknown user_id", goerr.V("user_id", userID))
	}

	return &genai.FunctionResponse{
		Name:     fc.Name,
		Response: userInfo,
	}, nil
}
