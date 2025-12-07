package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/urfave/cli/v3"
)

func similarCommand() *cli.Command {
	var (
		cfg       config
		alertID   string
		limit     int64
		threshold float64
		filters   []string
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "alert-id",
			Aliases:     []string{"i"},
			Usage:       "Alert ID to find similar alerts",
			Sources:     cli.EnvVars("LEVERET_ALERT_ID"),
			Destination: &alertID,
			Required:    true,
		},
		&cli.IntFlag{
			Name:        "limit",
			Aliases:     []string{"l"},
			Usage:       "Maximum number of similar alerts to display (applied after filtering)",
			Value:       10,
			Sources:     cli.EnvVars("LEVERET_SIMILAR_LIMIT"),
			Destination: &limit,
		},
		&cli.FloatFlag{
			Name:        "threshold",
			Aliases:     []string{"t"},
			Usage:       "Cosine distance threshold (0.0-2.0, lower is more similar)",
			Value:       1.0,
			Sources:     cli.EnvVars("LEVERET_SIMILAR_THRESHOLD"),
			Destination: &threshold,
		},
		&cli.StringSliceFlag{
			Name:        "filter",
			Aliases:     []string{"f"},
			Usage:       "Keyword filters (searches in title and description, multiple filters are AND-combined)",
			Sources:     cli.EnvVars("LEVERET_SIMILAR_FILTER"),
			Destination: &filters,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)

	return &cli.Command{
		Name:  "similar",
		Usage: "Find similar alerts using vector similarity based on alert ID",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			// Initialize repository
			repo, err := cfg.newRepository()
			if err != nil {
				return err
			}

			// Get the source alert
			sourceAlert, err := repo.GetAlert(ctx, model.AlertID(alertID))
			if err != nil {
				return goerr.Wrap(err, "failed to get source alert")
			}

			if len(sourceAlert.Embedding) == 0 {
				return goerr.New("source alert does not have an embedding vector")
			}

			// Search for similar alerts with threshold
			similarAlerts, err := repo.SearchSimilarAlerts(ctx, sourceAlert.Embedding, threshold)
			if err != nil {
				return goerr.Wrap(err, "failed to search similar alerts")
			}

			// Filter alerts
			var filtered []*model.Alert

			for _, alert := range similarAlerts {
				// Skip the source alert itself
				if alert.ID == sourceAlert.ID {
					continue
				}

				// Apply keyword filters (AND condition) on alert data
				if len(filters) > 0 {
					// Marshal alert data to JSON for filtering
					dataJSON, err := json.Marshal(alert.Data)
					if err != nil {
						return goerr.Wrap(err, "failed to marshal alert data", goerr.Value("alert_id", alert.ID))
					}
					dataStr := string(dataJSON)

					allMatch := true
					for _, filter := range filters {
						if !strings.Contains(dataStr, filter) {
							allMatch = false
							break
						}
					}
					if !allMatch {
						continue
					}
				}

				filtered = append(filtered, alert)
			}

			// Apply limit
			if int64(len(filtered)) > limit {
				filtered = filtered[:limit]
			}

			// Display results
			if len(filtered) == 0 {
				fmt.Fprintf(c.Root().Writer, "No similar alerts found\n")
				return nil
			}

			fmt.Fprintf(c.Root().Writer, "Found %d similar alerts for %s (%s):\n\n", len(filtered), sourceAlert.ID, sourceAlert.Title)
			for i, alert := range filtered {
				fmt.Fprintf(c.Root().Writer, "%d. %s\n", i+1, alert.ID)
				fmt.Fprintf(c.Root().Writer, "   Title: %s\n", alert.Title)
				if alert.Description != "" {
					fmt.Fprintf(c.Root().Writer, "   Description: %s\n", alert.Description)
				}
				fmt.Fprintf(c.Root().Writer, "\n")
			}

			return nil
		},
	}
}

