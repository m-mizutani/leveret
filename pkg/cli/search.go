package cli

import (
	"context"

	"github.com/urfave/cli/v3"
)

func searchCommand() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search for similar alerts using vector similarity",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "query",
				Aliases:  []string{"q"},
				Usage:    "Natural language query to search for similar alerts",
				Required: true,
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Usage:   "Maximum number of similar alerts to return",
				Value:   10,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// TODO: Implement search command
			return nil
		},
	}
}
