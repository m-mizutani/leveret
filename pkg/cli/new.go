package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/agent/bigquery"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/tool"
	toolAlert "github.com/m-mizutani/leveret/pkg/tool/alert"
	"github.com/m-mizutani/leveret/pkg/tool/otx"
	"github.com/m-mizutani/leveret/pkg/usecase/alert"
	"github.com/m-mizutani/leveret/pkg/workflow"
	"github.com/urfave/cli/v3"
)

func newCommand() *cli.Command {
	var (
		cfg       config
		mcpCfg    mcpConfig
		inputPath string
		policyDir string
	)

	// Create tool registry
	registry := tool.New(
		toolAlert.NewSearchAlerts(),
		otx.New(),
		bigquery.New(),
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "input",
			Aliases:     []string{"i"},
			Usage:       "Path to JSON file containing alert data",
			Sources:     cli.EnvVars("LEVERET_INPUT"),
			Destination: &inputPath,
		},
		&cli.StringFlag{
			Name:        "policy-dir",
			Usage:       "Directory containing Rego policy files",
			Sources:     cli.EnvVars("LEVERET_POLICY_DIR"),
			Destination: &policyDir,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)
	flags = append(flags, mcpFlags(&mcpCfg)...)
	flags = append(flags, registry.Flags()...)

	return &cli.Command{
		Name:  "new",
		Usage: "Create a new alert from JSON input",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			if inputPath == "" {
				return goerr.New("input file path is required")
			}

			// Read JSON file
			data, err := os.ReadFile(inputPath)
			if err != nil {
				return goerr.Wrap(err, "failed to read input file", goerr.Value("path", inputPath))
			}

			var alertData any
			if err := json.Unmarshal(data, &alertData); err != nil {
				return goerr.Wrap(err, "failed to parse JSON")
			}

			// Initialize dependencies
			repo, err := cfg.newRepository()
			if err != nil {
				return err
			}

			gemini, err := cfg.newGemini(ctx)
			if err != nil {
				return err
			}

			// Check if policy directory is specified
			if policyDir != "" {
				// Workflow mode: use OPA/Rego policies
				storage, err := cfg.newStorage(ctx)
				if err != nil {
					return err
				}

				// Load and initialize MCP if configured
				mcpProvider, err := mcpCfg.newMCP(ctx)
				if err != nil {
					return goerr.Wrap(err, "failed to initialize MCP")
				}

				// Add MCP provider to registry if available
				if mcpProvider != nil {
					registry.AddTool(mcpProvider)
				}

				// Initialize tools with client
				if err := registry.Init(ctx, &tool.Client{
					Repo:    repo,
					Gemini:  gemini,
					Storage: storage,
				}); err != nil {
					return goerr.Wrap(err, "failed to initialize tools")
				}

				engine, err := workflow.New(ctx, policyDir, gemini, registry)
				if err != nil {
					return goerr.Wrap(err, "failed to create workflow engine")
				}

				results, err := engine.Execute(ctx, alertData)
				if err != nil {
					return goerr.Wrap(err, "failed to execute workflow")
				}

				if len(results) == 0 {
					fmt.Fprintf(c.Root().Writer, "No alerts generated (rejected by ingest policy)\n")
					return nil
				}

				// Process workflow results
				for _, result := range results {
					fmt.Fprintf(c.Root().Writer, "Alert: %s\n", result.Alert.Title)

					// Check triage action
					if result.Triage != nil {
						fmt.Fprintf(c.Root().Writer, "  Action: %s, Severity: %s\n", result.Triage.Action, result.Triage.Severity)
						if result.Triage.Note != "" {
							fmt.Fprintf(c.Root().Writer, "  Note: %s\n", result.Triage.Note)
						}

						// Handle based on triage action
						switch result.Triage.Action {
						case "discard":
							fmt.Fprintf(c.Root().Writer, "  → Discarded (not saving to database)\n")
							continue
						case "notify":
							fmt.Fprintf(c.Root().Writer, "  → Notification mode (saving with lower priority)\n")
						case "accept":
							fmt.Fprintf(c.Root().Writer, "  → Accepted (saving to database)\n")
						default:
							fmt.Fprintf(c.Root().Writer, "  → Unknown action '%s' (treating as accept)\n", result.Triage.Action)
						}
					}

					// Generate alert ID and save to repository
					if result.Alert.ID == "" {
						result.Alert.ID = model.NewAlertID()
					}
					if result.Alert.CreatedAt.IsZero() {
						result.Alert.CreatedAt = time.Now()
					}
					if err := repo.PutAlert(ctx, result.Alert); err != nil {
						return goerr.Wrap(err, "failed to save alert")
					}

					fmt.Fprintf(c.Root().Writer, "  Alert ID: %s\n", result.Alert.ID)
				}
			} else {
				// Legacy mode: direct insert without workflow
				uc := alert.New(repo, gemini)
				newAlert, err := uc.Insert(ctx, alertData)
				if err != nil {
					return goerr.Wrap(err, "failed to insert alert")
				}

				fmt.Fprintf(c.Root().Writer, "Alert created: %s\n", newAlert.ID)
			}

			return nil
		},
	}
}
