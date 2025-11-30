package otx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

const otxBaseURL = "https://otx.alienvault.com/api/v1"

type queryOTXInput struct {
	IndicatorType string `json:"indicator_type"`
	Indicator     string `json:"indicator"`
	Section       string `json:"section"`
}

// getValidSections returns valid sections for a given indicator type
func getValidSections(indicatorType string) map[string]bool {
	switch indicatorType {
	case "IPv4", "IPv6":
		return map[string]bool{
			"general": true, "reputation": true, "geo": true, "malware": true,
			"url_list": true, "passive_dns": true, "http_scans": true,
		}
	case "domain", "hostname":
		return map[string]bool{
			"general": true, "geo": true, "malware": true,
			"url_list": true, "passive_dns": true, "whois": true,
		}
	case "file":
		return map[string]bool{
			"general": true, "analysis": true,
		}
	default:
		return map[string]bool{}
	}
}

// getSectionsList returns a list of valid sections for a given indicator type
func getSectionsList(indicatorType string) []string {
	sections := getValidSections(indicatorType)
	result := make([]string, 0, len(sections))
	for section := range sections {
		result = append(result, section)
	}
	return result
}

func (q *queryOTXInput) Validate() error {
	// Validate indicator_type
	validTypes := map[string]bool{
		"IPv4": true, "IPv6": true, "domain": true, "hostname": true, "file": true,
	}
	if !validTypes[q.IndicatorType] {
		return goerr.New("invalid indicator_type",
			goerr.V("indicator_type", q.IndicatorType),
			goerr.V("valid_types", []string{"IPv4", "IPv6", "domain", "hostname", "file"}))
	}

	// Validate indicator is not empty
	if q.Indicator == "" {
		return goerr.New("indicator is required")
	}

	// Validate section based on indicator type
	validSections := getValidSections(q.IndicatorType)
	if !validSections[q.Section] {
		return goerr.New("invalid section for indicator type",
			goerr.V("section", q.Section),
			goerr.V("indicator_type", q.IndicatorType),
			goerr.V("valid_sections", getSectionsList(q.IndicatorType)))
	}

	return nil
}

type otx struct {
	apiKey string
}

// New creates a new OTX tool
func New() *otx {
	return &otx{}
}

// Flags returns CLI flags for this tool
func (x *otx) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "otx-api-key",
			Sources:     cli.EnvVars("LEVERET_OTX_API_KEY"),
			Usage:       "OTX API key",
			Destination: &x.apiKey,
		},
	}
}

// Init initializes the tool
func (x *otx) Init(ctx context.Context, client *tool.Client) (bool, error) {
	// Only enable if API key is provided
	return x.apiKey != "", nil
}

// Prompt returns additional information to be added to the system prompt
func (x *otx) Prompt(ctx context.Context) string {
	return `### OTX Threat Intelligence

When analyzing security indicators (IP addresses, domains, file hashes, etc.), you can use the **query_otx** tool to get threat intelligence from AlienVault OTX.`
}

// Spec returns the tool specification for Gemini function calling
func (x *otx) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "query_otx",
				Description: "Query AlienVault OTX for threat intelligence about IP addresses, domains, hostnames, or file hashes. Available sections depend on indicator type: IPv4/IPv6 (general, reputation, geo, malware, url_list, passive_dns, http_scans), domain/hostname (general, geo, malware, url_list, passive_dns, whois), file (general, analysis)",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"indicator_type": {
							Type:        genai.TypeString,
							Description: "Type of indicator to query",
							Enum:        []string{"IPv4", "IPv6", "domain", "hostname", "file"},
						},
						"indicator": {
							Type:        genai.TypeString,
							Description: "The indicator value (IP address, domain, hostname, or file hash)",
						},
						"section": {
							Type:        genai.TypeString,
							Description: "Section of data to retrieve. Valid sections vary by indicator_type. Use 'general' for basic info that works with all types.",
						},
					},
					Required: []string{"indicator_type", "indicator", "section"},
				},
			},
		},
	}
}

// Execute runs the tool with the given function call
func (x *otx) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	// Marshal function call arguments to JSON
	paramsJSON, err := json.Marshal(fc.Args)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal function arguments")
	}

	var input queryOTXInput
	if err := json.Unmarshal(paramsJSON, &input); err != nil {
		return nil, goerr.Wrap(err, "failed to parse input parameters")
	}

	// Validate input
	if err := input.Validate(); err != nil {
		return nil, goerr.Wrap(err, "validation failed")
	}

	// Show progress
	fmt.Printf("🔍 OTX照会中: %s (%s) - %s セクション\n", input.Indicator, input.IndicatorType, input.Section)

	// Query OTX API
	result, err := x.queryAPI(ctx, input.IndicatorType, input.Indicator, input.Section)
	if err != nil {
		// Display detailed error information
		fmt.Printf("❌ OTXエラー: %v\n", err)
		if e, ok := err.(*goerr.Error); ok {
			if status, ok := e.Values()["status"]; ok {
				fmt.Printf("   ステータスコード: %v\n", status)
			}
			if body, ok := e.Values()["body"]; ok {
				fmt.Printf("   レスポンスボディ: %v\n", body)
			}
		}
		return nil, goerr.Wrap(err, "failed to query OTX API")
	}

	// Convert result to JSON string for better readability
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal result")
	}

	return &genai.FunctionResponse{
		Name:     fc.Name,
		Response: map[string]any{"result": string(resultJSON)},
	}, nil
}

// queryAPI queries OTX API for a specific indicator and section
func (x *otx) queryAPI(ctx context.Context, indicatorType, indicator, section string) (map[string]any, error) {
	url := fmt.Sprintf("%s/indicators/%s/%s/%s", otxBaseURL, indicatorType, indicator, section)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("X-OTX-API-KEY", x.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, goerr.New("OTX API returned error",
			goerr.V("status", resp.StatusCode),
			goerr.V("body", string(body)))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, goerr.Wrap(err, "failed to decode response")
	}

	return result, nil
}
