package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

type Error struct {
	Code    int
	Message string
}

func Run(ctx context.Context, argv []string) *Error {
	var (
		logLevel string
		verbose  bool
		logger   *slog.Logger
	)

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
			historyCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Aliases:     []string{"l"},
				Usage:       "Log level (debug, info, warn, error)",
				Value:       "info",
				Sources:     cli.EnvVars("LEVERET_LOG_LEVEL"),
				Destination: &logLevel,
			},
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Usage:       "Enable verbose mode (show stack traces on error)",
				Sources:     cli.EnvVars("LEVERET_VERBOSE"),
				Destination: &verbose,
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			logger = logging.New(logLevel, os.Stderr)
			logging.SetDefault(logger)
			return logging.With(ctx, logger), nil
		},
	}

	if err := cmd.Run(ctx, argv); err != nil {
		var logAttrs []any
		if goErr := goerr.Unwrap(err); goErr != nil {
			var attrs []any
			if values := goErr.Values(); len(values) > 0 {
				attrs = append(attrs, slog.Any("values", values))
			}
			if tags := goErr.Tags(); len(tags) > 0 {
				attrs = append(attrs, slog.Any("tags", tags))
			}
			if tv := goErr.TypedValues(); len(tv) > 0 {
				attrs = append(attrs, slog.Any("typedValues", tv))
			}

			logAttrs = append(logAttrs, attrs...)
		}

		logger.Error(err.Error(), logAttrs...)

		if goErr := goerr.Unwrap(err); goErr != nil && verbose {
			fmt.Fprintf(os.Stderr, "\n%+v\n", err)
		}

		return &Error{
			Code:    1,
			Message: err.Error(),
		}
	}

	return nil
}
