package bigquery

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

// Tool is the BigQuery sub-agent tool that provides natural language query interface
type Tool struct {
	// Configuration
	project         string
	configFile      string
	runBookDir      string
	scanLimitMB     int64
	resultLimitRows int64

	// Dependencies
	bq      adapter.BigQuery
	gemini  adapter.Gemini
	agent   *Agent
	enabled bool
	output  io.Writer
}

// New creates a new BigQuery sub-agent tool
func New() tool.Tool {
	return &Tool{
		scanLimitMB:     1024,
		resultLimitRows: 1000,
		output:          os.Stdout,
	}
}

// Flags returns CLI flags for BigQuery sub-agent
func (t *Tool) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "bigquery-project",
			Usage:       "Google Cloud project ID for BigQuery",
			Sources:     cli.EnvVars("LEVERET_BIGQUERY_PROJECT"),
			Destination: &t.project,
		},
		&cli.StringFlag{
			Name:        "bigquery-runbook-dir",
			Usage:       "Directory containing SQL runbook files",
			Sources:     cli.EnvVars("LEVERET_BIGQUERY_RUNBOOK_DIR"),
			Destination: &t.runBookDir,
		},
		&cli.StringFlag{
			Name:        "bigquery-config-file",
			Usage:       "Configuration file containing BigQuery table definitions",
			Sources:     cli.EnvVars("LEVERET_BIGQUERY_CONFIG_FILE"),
			Destination: &t.configFile,
		},
		&cli.IntFlag{
			Name:        "bigquery-scan-limit-mb",
			Usage:       "Maximum scan limit in MB for dry-run validation",
			Value:       10240,
			Sources:     cli.EnvVars("LEVERET_BIGQUERY_SCAN_LIMIT_MB"),
			Destination: &t.scanLimitMB,
		},
		&cli.IntFlag{
			Name:        "bigquery-result-limit-rows",
			Usage:       "Maximum number of rows to return per query result request",
			Value:       1000,
			Sources:     cli.EnvVars("LEVERET_BIGQUERY_RESULT_LIMIT_ROWS"),
			Destination: &t.resultLimitRows,
		},
	}
}

// Init initializes the BigQuery sub-agent
func (t *Tool) Init(ctx context.Context, client *tool.Client) (bool, error) {
	if t.project == "" {
		return false, nil
	}

	// Initialize BigQuery client
	bq, err := adapter.NewBigQuery(ctx, t.project)
	if err != nil {
		return false, goerr.Wrap(err, "failed to create BigQuery client")
	}
	t.bq = bq
	t.gemini = client.Gemini

	// Load runBooks if directory is configured
	var runBooks map[string]*runBook
	if t.runBookDir != "" {
		runBooks, err = loadRunBooks(t.runBookDir)
		if err != nil {
			return false, goerr.Wrap(err, "failed to load runBooks")
		}
	}

	// Load table list from config file if configured
	var tables []tableInfo
	if t.configFile != "" {
		tables, err = loadTableList(t.configFile)
		if err != nil {
			return false, goerr.Wrap(err, "failed to load table list")
		}
	}

	// Create the agent
	t.agent = NewAgent(
		t.gemini,
		t.bq,
		WithRunBooks(runBooks),
		WithTables(tables),
		WithScanLimitMB(t.scanLimitMB),
		WithResultLimitRows(t.resultLimitRows),
		WithOutput(t.output),
	)

	t.enabled = true
	return true, nil
}

// Prompt returns additional information to be added to the system prompt
func (t *Tool) Prompt(ctx context.Context) string {
	if !t.enabled || t.agent == nil {
		return ""
	}

	// Only expose table list to main agent (not runbooks)
	if len(t.agent.tables) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("### BigQuery Tables\n\n")
	for _, table := range t.agent.tables {
		sb.WriteString(fmt.Sprintf("- **%s**", table.FullName()))
		if table.Description != "" {
			sb.WriteString(fmt.Sprintf(": %s", table.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// Spec returns the tool specification for Gemini function calling
func (t *Tool) Spec() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "bigquery_run",
				Description: "Execute BigQuery analysis using natural language. This tool translates your request into SQL queries, executes them, and returns analysis results. Use this for log analysis, security investigation, and data exploration.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"query": {
							Type:        genai.TypeString,
							Description: "Natural language description of what you want to analyze or query from BigQuery",
						},
					},
					Required: []string{"query"},
				},
			},
		},
	}
}

// Execute runs the BigQuery sub-agent with the given natural language query
func (t *Tool) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	if fc.Name != "bigquery_run" {
		return nil, goerr.New("unknown function", goerr.V("name", fc.Name))
	}

	queryStr, ok := fc.Args["query"].(string)
	if !ok || queryStr == "" {
		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"error": "query parameter is required",
			},
		}, nil
	}

	// Execute the sub-agent
	result, err := t.agent.Execute(ctx, queryStr)
	if err != nil {
		return &genai.FunctionResponse{
			Name: fc.Name,
			Response: map[string]any{
				"error": err.Error(),
			},
		}, nil
	}

	return &genai.FunctionResponse{
		Name: fc.Name,
		Response: map[string]any{
			"result": result,
		},
	}, nil
}
