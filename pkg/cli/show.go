package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/usecase/alert"
	"github.com/urfave/cli/v3"
)

func showCommand() *cli.Command {
	var (
		cfg     config
		alertID model.AlertID
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "alert-id",
			Aliases:     []string{"id"},
			Usage:       "Alert ID to show",
			Sources:     cli.EnvVars("LEVERET_ALERT_ID"),
			Destination: (*string)(&alertID),
			Required:    true,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)

	return &cli.Command{
		Name:  "show",
		Usage: "Show detailed information of a specific alert",
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

			// Create alert usecase
			uc := alert.New(repo, gemini)

			// Show alert
			a, err := uc.Show(ctx, alertID)
			if err != nil {
				return goerr.Wrap(err, "failed to show alert")
			}

			// Display alert details
			data, err := json.MarshalIndent(a, "", "  ")
			if err != nil {
				return goerr.Wrap(err, "failed to marshal alert")
			}

			fmt.Fprintf(c.Root().Writer, "%s\n", string(data))
			return nil
		},
	}
}
