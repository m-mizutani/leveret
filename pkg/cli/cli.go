package cli

import (
	"context"

	"github.com/urfave/cli/v3"
)

type Error struct {
	Code    int
	Message string
}

func Run(ctx context.Context, argv []string) *Error {
	cmd := &cli.Command{
		Name:  "leveret",
		Usage: "Security alert analysis agent",
		Commands: []*cli.Command{
			newCommand(),
			chatCommand(),
			listCommand(),
			resolveCommand(),
			mergeCommand(),
			unmergeCommand(),
		},
	}

	if err := cmd.Run(ctx, argv); err != nil {
		return &Error{
			Code:    1,
			Message: err.Error(),
		}
	}

	return nil
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:  "new",
		Usage: "Create a new alert from JSON input",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "input",
				Aliases: []string{"i"},
				Usage:   "Path to JSON file containing alert data",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// TODO: Implement new command
			return nil
		},
	}
}

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

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all alerts",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Include merged alerts",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// TODO: Implement list command
			return nil
		},
	}
}

func resolveCommand() *cli.Command {
	return &cli.Command{
		Name:      "resolve",
		Usage:     "Mark an alert as resolved",
		ArgsUsage: "<alert-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "conclusion",
				Aliases: []string{"c"},
				Usage:   "Conclusion message",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// TODO: Implement resolve command
			return nil
		},
	}
}

func mergeCommand() *cli.Command {
	return &cli.Command{
		Name:      "merge",
		Usage:     "Merge an alert into another",
		ArgsUsage: "<source-id> <target-id>",
		Action: func(ctx context.Context, c *cli.Command) error {
			// TODO: Implement merge command
			return nil
		},
	}
}

func unmergeCommand() *cli.Command {
	return &cli.Command{
		Name:      "unmerge",
		Usage:     "Unmerge a merged alert",
		ArgsUsage: "<alert-id>",
		Action: func(ctx context.Context, c *cli.Command) error {
			// TODO: Implement unmerge command
			return nil
		},
	}
}
