package cli

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/briandowns/spinner"
	"github.com/chzyer/readline"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/agent/bigquery"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/m-mizutani/leveret/pkg/tool/alert"
	"github.com/m-mizutani/leveret/pkg/tool/otx"
	"github.com/m-mizutani/leveret/pkg/usecase/chat"
	"github.com/urfave/cli/v3"
)

func chatCommand() *cli.Command {
	var (
		cfg             config
		mcpCfg          mcpConfig
		alertID         model.AlertID
		environmentInfo string
	)

	// Create tool registry early to get flags
	registry := tool.New(
		alert.NewSearchAlerts(),
		otx.New(),
		bigquery.New(),
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "alert-id",
			Aliases:     []string{"i"},
			Usage:       "Alert ID to chat with",
			Sources:     cli.EnvVars("LEVERET_ALERT_ID"),
			Destination: (*string)(&alertID),
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "environment-info",
			Usage:       "Environment context information for better analysis",
			Sources:     cli.EnvVars("LEVERET_ENVIRONMENT_INFO"),
			Destination: &environmentInfo,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)
	flags = append(flags, mcpFlags(&mcpCfg)...)
	flags = append(flags, registry.Flags()...)

	return &cli.Command{
		Name:  "chat",
		Usage: "Interactive analysis of an alert",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			// Initialize dependencies
			repo, err := cfg.newRepository()
			if err != nil {
				return err
			}

			gemini, err := cfg.newGemini(ctx)
			if err != nil {
				return err
			}

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

			// Display enabled tools
			if enabledTools := registry.EnabledTools(); len(enabledTools) > 0 {
				fmt.Printf("Enabled tools: %v\n", enabledTools)
			} else {
				fmt.Printf("No tools enabled\n")
			}

			// Create chat session
			session, err := chat.New(ctx, chat.NewInput{
				Repo:            repo,
				Gemini:          gemini,
				Storage:         storage,
				Registry:        registry,
				AlertID:         alertID,
				EnvironmentInfo: environmentInfo,
			})
			if err != nil {
				return goerr.Wrap(err, "failed to create chat session")
			}

			fmt.Printf("\n")
			// Interactive chat loop with readline support
			rl, err := readline.New("> ")
			if err != nil {
				return goerr.Wrap(err, "failed to create readline")
			}
			defer rl.Close()

			fmt.Fprintf(c.Root().Writer, "Chat session started. Type 'exit' to quit.\n\n")

			for {
				line, err := rl.Readline()
				if err != nil {
					if err == io.EOF || err == readline.ErrInterrupt {
						break
					}
					return goerr.Wrap(err, "failed to read line")
				}

				message := line
				if message == "exit" {
					break
				}

				if message == "" {
					continue
				}

				fmt.Fprintf(c.Root().Writer, "\n")

				// Start spinner with random words
				words := []string{"analyzing", "processing", "thinking", "searching", "evaluating", "examining", "investigating", "reviewing"}
				s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
				s.Suffix = " " + words[rand.Intn(len(words))] + "..."
				s.Start()

				// Send message to Gemini
				response, err := session.Send(ctx, message)
				s.Stop()

				if err != nil {
					return goerr.Wrap(err, "failed to send message")
				}

				// Display response
				if response != nil {
					for _, candidate := range response.Candidates {
						if candidate.Content != nil {
							for _, part := range candidate.Content.Parts {
								if text := part.Text; text != "" {
									fmt.Fprintf(c.Root().Writer, "%s\n", text)
								}
							}
						}
					}
					fmt.Fprintf(c.Root().Writer, "\n")
				}
			}

			fmt.Fprintf(c.Root().Writer, "\nChat session completed\n")
			return nil
		},
	}
}
