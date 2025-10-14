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
			showCommand(),
			searchCommand(),
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
