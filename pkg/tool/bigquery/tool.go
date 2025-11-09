package bigquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

// Tool is the BigQuery tool that provides 4 functions
type Tool struct {
	// Configuration
	project         string
	configFile      string
	runBookDir      string
	scanLimitMB     int64
	resultLimitRows int64

	// Dependencies
	bq       adapter.BigQuery
	runBooks map[string]*runBook
	tables   []tableInfo

	// In-memory result storage (jobID -> results)
	results map[string][]map[string]any
}

// New creates a new BigQuery tool
func New() tool.Tool {
	return &Tool{
		scanLimitMB:     1024, // Default 1GB
		resultLimitRows: 1000, // Default 1000 rows
		results:         make(map[string][]map[string]any),
	}
}

// Flags returns CLI flags for BigQuery tools
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

// Init initializes the BigQuery tool
func (t *Tool) Init(ctx context.Context, client *tool.Client) (bool, error) {
	// BigQuery is optional - only enable if project is configured
	if t.project == "" {
		return false, nil
	}

	// Initialize BigQuery client
	bq, err := adapter.NewBigQuery(ctx, t.project)
	if err != nil {
		return false, goerr.Wrap(err, "failed to create BigQuery client")
	}
	t.bq = bq

	// Load runBooks if directory is configured
	if t.runBookDir != "" {
		runBooks, err := loadRunBooks(t.runBookDir)
		if err != nil {
			return false, goerr.Wrap(err, "failed to load runBooks")
		}
		t.runBooks = runBooks
	}

	// Load table list from config file if configured
	if t.configFile != "" {
		tables, err := loadTableList(t.configFile)
		if err != nil {
			return false, goerr.Wrap(err, "failed to load table list")
		}
		t.tables = tables
	}

	return true, nil
}

// Prompt returns additional information to be added to the system prompt
func (t *Tool) Prompt(ctx context.Context) string {
	var lines []string

	// Add runBook information
	if len(t.runBooks) > 0 {
		lines = append(lines, "### BigQuery RunBooks")
		lines = append(lines, "")
		for _, rb := range t.runBooks {
			line := fmt.Sprintf("- **ID**: `%s`", rb.ID)
			if rb.Title != "" {
				line += fmt.Sprintf(", **Title**: %s", rb.Title)
			}
			if rb.Description != "" {
				line += fmt.Sprintf(", **Description**: %s", rb.Description)
			}
			lines = append(lines, line)
		}
	}

	// Add table list information
	if len(t.tables) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "### BigQuery Tables")
		lines = append(lines, "")
		for _, table := range t.tables {
			line := fmt.Sprintf("- **%s**", table.FullName())
			if table.Description != "" {
				line += fmt.Sprintf(": %s", table.Description)
			}
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n")
}

// Spec returns the tool specification for Gemini function calling
func (t *Tool) Spec() *genai.Tool {
	declarations := []*genai.FunctionDeclaration{
		{
			Name:        "bigquery_query",
			Description: fmt.Sprintf("Execute a BigQuery SQL query with automatic dry-run validation. The query is validated for scan limit (max: %d MB) before execution and results are stored for later retrieval.", t.scanLimitMB),
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"query": {
						Type:        genai.TypeString,
						Description: "SQL query to execute",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "bigquery_get_result",
			Description: "Get results from a previously executed BigQuery job with pagination support",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"job_id": {
						Type:        genai.TypeString,
						Description: "Job ID returned from bigquery_query",
					},
					"limit": {
						Type:        genai.TypeInteger,
						Description: fmt.Sprintf("Maximum number of rows to return (default: 100, max: %d)", t.resultLimitRows),
					},
					"offset": {
						Type:        genai.TypeInteger,
						Description: "Number of rows to skip for pagination (default: 0, must be >= 0)",
					},
				},
				Required: []string{"job_id"},
			},
		},
		{
			Name:        "bigquery_schema",
			Description: "Get schema information for a BigQuery table including field names, types, and descriptions",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"project": {
						Type:        genai.TypeString,
						Description: "Google Cloud project ID",
					},
					"dataset_id": {
						Type:        genai.TypeString,
						Description: "BigQuery dataset ID",
					},
					"table": {
						Type:        genai.TypeString,
						Description: "BigQuery table name",
					},
				},
				Required: []string{"project", "dataset_id", "table"},
			},
		},
	}

	// Add bigquery_runbook if runBooks are loaded
	if len(t.runBooks) > 0 {
		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:        "bigquery_runbook",
			Description: "Get SQL query from runBook by ID",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"runbook_id": {
						Type:        genai.TypeString,
						Description: "RunBook ID to retrieve",
					},
				},
				Required: []string{"runbook_id"},
			},
		})
	}

	return &genai.Tool{
		FunctionDeclarations: declarations,
	}
}

// Execute runs the tool with the given function call
func (t *Tool) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	switch fc.Name {
	case "bigquery_query":
		return t.executeQuery(ctx, fc)
	case "bigquery_get_result":
		return t.executeGetResult(ctx, fc)
	case "bigquery_schema":
		return t.executeSchema(ctx, fc)
	case "bigquery_runbook":
		return t.executeRunbook(ctx, fc)
	default:
		return nil, goerr.New("unknown function", goerr.V("name", fc.Name))
	}
}
