package otx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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

type otx struct {
	apiKey     string
	httpClient *http.Client
}

// New creates a new OTX tool
func New() *otx {
	return &otx{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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
	return `When analyzing security indicators (IP addresses, domains, file hashes, etc.), you can use the query_otx tool to get threat intelligence from AlienVault OTX.`
}

// Spec returns the tool specification for Gemini function calling
func (x *otx) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "query_otx",
				Description: "Query AlienVault OTX for threat intelligence about IP addresses, domains, hostnames, or file hashes",
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
							Description: "Section of data to retrieve",
							Enum:        []string{"general", "reputation", "geo", "malware", "url_list", "passive_dns", "http_scans", "nids_list", "analysis", "whois"},
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

	// Query OTX API
	result, err := x.queryAPI(ctx, input.IndicatorType, input.Indicator, input.Section)
	if err != nil {
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

	resp, err := x.httpClient.Do(req)
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
