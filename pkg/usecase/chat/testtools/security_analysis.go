package testtools

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

// getAlertDetailsTool returns alert details
type getAlertDetailsTool struct {
	callCount int
	alertIDs  []string
}

func NewGetAlertDetailsTool() *getAlertDetailsTool {
	return &getAlertDetailsTool{callCount: 0, alertIDs: []string{}}
}

func (t *getAlertDetailsTool) CallCount() int {
	return t.callCount
}

func (t *getAlertDetailsTool) CalledAlertIDs() []string {
	return t.alertIDs
}

func (t *getAlertDetailsTool) Flags() []cli.Flag {
	return nil
}

func (t *getAlertDetailsTool) Init(ctx context.Context, client *tool.Client) (bool, error) {
	return true, nil
}

func (t *getAlertDetailsTool) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_alert_details",
				Description: "Get detailed information about a specific alert",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"alert_id": {
							Type:        genai.TypeString,
							Description: "Alert ID to get details for",
						},
					},
					Required: []string{"alert_id"},
				},
			},
		},
	}
}

func (t *getAlertDetailsTool) Prompt(ctx context.Context) string {
	return ""
}

func (t *getAlertDetailsTool) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	t.callCount++

	alertID, ok := fc.Args["alert_id"].(string)
	if !ok {
		return nil, goerr.New("alert_id argument is required")
	}
	t.alertIDs = append(t.alertIDs, alertID)

	// Accept any alert ID and return test data
	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"alert_id":  alertID,
			"type":      "suspicious_login",
			"source_ip": "203.0.113.42",
			"timestamp": "2024-01-01T15:30:00Z",
		},
	}, nil
}

// checkIPReputationTool checks IP reputation
type checkIPReputationTool struct {
	callCount int
	ips       []string
}

func NewCheckIPReputationTool() *checkIPReputationTool {
	return &checkIPReputationTool{callCount: 0, ips: []string{}}
}

func (t *checkIPReputationTool) CallCount() int {
	return t.callCount
}

func (t *checkIPReputationTool) CalledIPs() []string {
	return t.ips
}

func (t *checkIPReputationTool) Flags() []cli.Flag {
	return nil
}

func (t *checkIPReputationTool) Init(ctx context.Context, client *tool.Client) (bool, error) {
	return true, nil
}

func (t *checkIPReputationTool) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "check_ip_reputation",
				Description: "Check the reputation of an IP address",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"ip": {
							Type:        genai.TypeString,
							Description: "IP address to check",
						},
					},
					Required: []string{"ip"},
				},
			},
		},
	}
}

func (t *checkIPReputationTool) Prompt(ctx context.Context) string {
	return ""
}

func (t *checkIPReputationTool) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	t.callCount++

	ip, ok := fc.Args["ip"].(string)
	if !ok {
		return nil, goerr.New("ip argument is required")
	}
	t.ips = append(t.ips, ip)

	if ip != "203.0.113.42" {
		return nil, goerr.New("unknown ip", goerr.V("ip", ip))
	}

	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"ip":               "203.0.113.42",
			"reputation_score": 25,
			"category":         "known_scanner",
			"last_seen":        "2024-01-01T12:00:00Z",
		},
	}, nil
}

// getHistoricalAlertsTool gets historical alerts for an IP
type getHistoricalAlertsTool struct {
	callCount int
	sourceIPs []string
}

func NewGetHistoricalAlertsTool() *getHistoricalAlertsTool {
	return &getHistoricalAlertsTool{callCount: 0, sourceIPs: []string{}}
}

func (t *getHistoricalAlertsTool) CallCount() int {
	return t.callCount
}

func (t *getHistoricalAlertsTool) CalledSourceIPs() []string {
	return t.sourceIPs
}

func (t *getHistoricalAlertsTool) Flags() []cli.Flag {
	return nil
}

func (t *getHistoricalAlertsTool) Init(ctx context.Context, client *tool.Client) (bool, error) {
	return true, nil
}

func (t *getHistoricalAlertsTool) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_historical_alerts",
				Description: "Get historical alerts from the same source IP",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"source_ip": {
							Type:        genai.TypeString,
							Description: "Source IP address to search for",
						},
					},
					Required: []string{"source_ip"},
				},
			},
		},
	}
}

func (t *getHistoricalAlertsTool) Prompt(ctx context.Context) string {
	return ""
}

func (t *getHistoricalAlertsTool) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	t.callCount++

	sourceIP, ok := fc.Args["source_ip"].(string)
	if !ok {
		return nil, goerr.New("source_ip argument is required")
	}
	t.sourceIPs = append(t.sourceIPs, sourceIP)

	if sourceIP != "203.0.113.42" {
		return nil, goerr.New("unknown source_ip", goerr.V("source_ip", sourceIP))
	}

	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"total_count": 15,
			"time_range":  "30_days",
			"alert_types": []string{"port_scan", "brute_force", "suspicious_login"},
		},
	}, nil
}
