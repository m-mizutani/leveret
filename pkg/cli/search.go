package cli

import (
	"context"
	"fmt"

	"github.com/m-mizutani/leveret/pkg/usecase/alert"
	"github.com/urfave/cli/v3"
)

func searchCommand() *cli.Command {
	var (
		cfg   config
		query string
		limit int64
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "query",
			Aliases:     []string{"q"},
			Usage:       "Natural language query to search for similar alerts",
			Sources:     cli.EnvVars("LEVERET_SEARCH_QUERY"),
			Destination: &query,
			Required:    true,
		},
		&cli.IntFlag{
			Name:        "limit",
			Aliases:     []string{"l"},
			Usage:       "Maximum number of similar alerts to return",
			Value:       10,
			Sources:     cli.EnvVars("LEVERET_SEARCH_LIMIT"),
			Destination: &limit,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)

	return &cli.Command{
		Name:  "search",
		Usage: "Search for similar alerts using vector similarity",
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
			uc := alert.New(repo, gemini, alert.WithOutput(c.Root().Writer))

			// Search for similar alerts
			alerts, err := uc.Search(ctx, alert.SearchOptions{
				Query: query,
				Limit: int(limit),
			})
			if err != nil {
				return err
			}

			// Display results
			if len(alerts) == 0 {
				fmt.Fprintf(c.Root().Writer, "No similar alerts found\n")
				return nil
			}

			fmt.Fprintf(c.Root().Writer, "Found %d similar alerts:\n\n", len(alerts))
			for i, a := range alerts {
				fmt.Fprintf(c.Root().Writer, "%d. %s (%s)\n", i+1, a.ID, a.Title)
				if a.Description != "" {
					fmt.Fprintf(c.Root().Writer, "   %s\n", a.Description)
				}
				fmt.Fprintf(c.Root().Writer, "\n")
			}

			return nil
		},
	}
}
