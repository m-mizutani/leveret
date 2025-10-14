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

			claude, err := cfg.newClaude()
			if err != nil {
				return err
			}

			gemini, err := cfg.newGemini()
			if err != nil {
				return err
			}

			// Create alert usecase
			uc := alert.New(repo, claude, gemini, alert.WithOutput(c.Root().Writer))

			// TODO: Implement Search method
			_ = uc
			_ = query
			_ = limit

			fmt.Fprintf(c.Root().Writer, "Search completed\n")
			return nil
		},
	}
}
