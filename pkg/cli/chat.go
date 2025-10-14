package cli

import (
	"context"
	"fmt"

	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/urfave/cli/v3"
)

func chatCommand() *cli.Command {
	var (
		cfg     config
		alertID string
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "alert-id",
			Aliases:     []string{"id"},
			Usage:       "Alert ID to chat with",
			Sources:     cli.EnvVars("LEVERET_ALERT_ID"),
			Destination: &alertID,
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

			claude, err := cfg.newClaude()
			if err != nil {
				return err
			}

			gemini, err := cfg.newGemini()
			if err != nil {
				return err
			}

			storage, err := cfg.newStorage(ctx)
			if err != nil {
				return err
			}

			// TODO: Implement chat.Start
			_ = repo
			_ = claude
			_ = gemini
			_ = storage
			_ = model.AlertID(alertID)

			fmt.Fprintf(c.Root().Writer, "Chat session completed\n")
			return nil
		},
	}
}
