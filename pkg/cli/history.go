package cli

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/urfave/cli/v3"
)

func historyCommand() *cli.Command {
	var (
		cfg     config
		alertID string
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "alert-id",
			Aliases:     []string{"i"},
			Usage:       "Alert ID to list conversation histories",
			Sources:     cli.EnvVars("LEVERET_ALERT_ID"),
			Destination: &alertID,
			Required:    true,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)

	return &cli.Command{
		Name:  "history",
		Usage: "List conversation histories for an alert",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			// Initialize repository
			repo, err := cfg.newRepository()
			if err != nil {
				return err
			}

			// List histories
			histories, err := repo.ListHistoryByAlert(ctx, model.AlertID(alertID))
			if err != nil {
				return goerr.Wrap(err, "failed to list histories")
			}

			// Display histories
			if len(histories) == 0 {
				fmt.Fprintf(c.Root().Writer, "No conversation histories found for alert %s\n", alertID)
				return nil
			}

			for _, h := range histories {
				fmt.Fprintf(c.Root().Writer, "%s\t%s\t%s\t%s\n",
					h.ID,
					h.Title,
					h.CreatedAt.Format("2006-01-02 15:04:05"),
					h.UpdatedAt.Format("2006-01-02 15:04:05"),
				)
			}

			return nil
		},
	}
}
