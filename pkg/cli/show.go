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
	var cfg config

	return &cli.Command{
		Name:      "show",
		Usage:     "Show detailed information of a specific alert",
		ArgsUsage: "<alert-id>",
		Flags:     append(globalFlags(&cfg), llmFlags(&cfg)...),
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() == 0 {
				return goerr.New("alert-id is required")
			}
			alertID := model.AlertID(c.Args().Get(0))

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

			// Create alert usecase
			uc := alert.New(repo, claude, gemini)

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
