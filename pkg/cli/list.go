package cli

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/usecase/alert"
	"github.com/urfave/cli/v3"
)

func listCommand() *cli.Command {
	var (
		cfg    config
		all    bool
		offset int64
		limit  int64
	)

	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:        "all",
			Aliases:     []string{"a"},
			Usage:       "Include merged alerts",
			Sources:     cli.EnvVars("LEVERET_LIST_ALL"),
			Destination: &all,
		},
		&cli.IntFlag{
			Name:        "offset",
			Usage:       "Offset for pagination",
			Value:       0,
			Sources:     cli.EnvVars("LEVERET_LIST_OFFSET"),
			Destination: &offset,
		},
		&cli.IntFlag{
			Name:        "limit",
			Usage:       "Maximum number of alerts to list",
			Value:       100,
			Sources:     cli.EnvVars("LEVERET_LIST_LIMIT"),
			Destination: &limit,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)

	return &cli.Command{
		Name:  "list",
		Usage: "List all alerts",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			// Initialize dependencies
			repo, err := cfg.newRepository()
			if err != nil {
				return err
			}

			gemini, err := cfg.newGemini()
			if err != nil {
				return err
			}

			// Create alert usecase
			uc := alert.New(repo, gemini)

			// List alerts
			alerts, err := uc.List(ctx, alert.ListOptions{
				IncludeMerged: all,
				Offset:        int(offset),
				Limit:         int(limit),
			})
			if err != nil {
				return goerr.Wrap(err, "failed to list alerts")
			}

			// Display alerts
			for _, a := range alerts {
				status := "active"
				if a.ResolvedAt != nil {
					status = "resolved"
				}
				if a.MergedTo != "" {
					status = fmt.Sprintf("merged to %s", a.MergedTo)
				}
				fmt.Fprintf(c.Root().Writer, "%s\t%s\t%s\n", a.ID, a.Title, status)
			}

			return nil
		},
	}
}
