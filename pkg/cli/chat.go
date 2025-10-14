package cli

import (
	"context"

	"github.com/urfave/cli/v3"
)

func chatCommand() *cli.Command {
	return &cli.Command{
		Name:      "chat",
		Usage:     "Interactive analysis of an alert",
		ArgsUsage: "<alert-id>",
		Action: func(ctx context.Context, c *cli.Command) error {
			// TODO: Implement chat command
			return nil
		},
	}
}
