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
	var (
		cfg      config
		sourceID model.AlertID
		targetID model.AlertID
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "source-id",
			Aliases:     []string{"s"},
			Usage:       "Source alert ID to merge from",
			Sources:     cli.EnvVars("LEVERET_MERGE_SOURCE_ID"),
			Destination: (*string)(&sourceID),
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "target-id",
			Aliases:     []string{"t"},
			Usage:       "Target alert ID to merge into",
			Sources:     cli.EnvVars("LEVERET_MERGE_TARGET_ID"),
			Destination: (*string)(&targetID),
			Required:    true,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)

	return &cli.Command{
		Name:  "merge",
		Usage: "Merge an alert into another",
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
	var (
		cfg     config
		alertID model.AlertID
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "alert-id",
			Aliases:     []string{"i"},
			Usage:       "Alert ID to unmerge",
			Sources:     cli.EnvVars("LEVERET_ALERT_ID"),
			Destination: (*string)(&alertID),
			Required:    true,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)

	return &cli.Command{
		Name:  "unmerge",
		Usage: "Unmerge a merged alert",
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

			// Unmerge alert
			if err := uc.Unmerge(ctx, alertID); err != nil {
				return goerr.Wrap(err, "failed to unmerge alert")
			}

			fmt.Fprintf(c.Root().Writer, "Alert unmerged: %s\n", alertID)
			return nil
		},
	}
}
