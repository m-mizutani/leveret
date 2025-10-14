package cli

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/usecase/alert"
	"github.com/urfave/cli/v3"
)

func mergeCommand() *cli.Command {
	var cfg config

	return &cli.Command{
		Name:      "merge",
		Usage:     "Merge an alert into another",
		ArgsUsage: "<source-id> <target-id>",
		Flags:     append(globalFlags(&cfg), llmFlags(&cfg)...),
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() < 2 {
				return goerr.New("source-id and target-id are required")
			}
			sourceID := model.AlertID(c.Args().Get(0))
			targetID := model.AlertID(c.Args().Get(1))

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

			// Merge alerts
			if err := uc.Merge(ctx, sourceID, targetID); err != nil {
				return goerr.Wrap(err, "failed to merge alerts")
			}

			fmt.Fprintf(c.Root().Writer, "Alert %s merged to %s\n", sourceID, targetID)
			return nil
		},
	}
}

func unmergeCommand() *cli.Command {
	var cfg config

	return &cli.Command{
		Name:      "unmerge",
		Usage:     "Unmerge a merged alert",
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

			// Unmerge alert
			if err := uc.Unmerge(ctx, alertID); err != nil {
				return goerr.Wrap(err, "failed to unmerge alert")
			}

			fmt.Fprintf(c.Root().Writer, "Alert unmerged: %s\n", alertID)
			return nil
		},
	}
}
