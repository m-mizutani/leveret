package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/usecase/chat"
	"github.com/urfave/cli/v3"
)

func chatCommand() *cli.Command {
	var (
		cfg     config
		alertID model.AlertID
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "alert-id",
			Aliases:     []string{"id"},
			Usage:       "Alert ID to chat with",
			Sources:     cli.EnvVars("LEVERET_ALERT_ID"),
			Destination: (*string)(&alertID),
			Required:    true,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)

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

			// Create chat session
			session, err := chat.New(ctx, chat.NewInput{
				Repo:    repo,
				Gemini:  gemini,
				Storage: storage,
				AlertID: alertID,
			})
			if err != nil {
				return goerr.Wrap(err, "failed to create chat session")
			}

			// Interactive chat loop
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Fprintf(c.Root().Writer, "Chat session started. Type 'exit' to quit.\n")

			for {
				fmt.Fprintf(c.Root().Writer, "> ")
				if !scanner.Scan() {
					break
				}

				message := scanner.Text()
				if message == "exit" {
					break
				}

				if message == "" {
					continue
				}

				// Send message to Gemini
				response, err := session.Send(ctx, message)
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
				}
			}

			fmt.Fprintf(c.Root().Writer, "\nChat session completed\n")
			return nil
		},
	}
}
