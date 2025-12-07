package cli

import (
	"context"
	"fmt"
	"math"
	"strings"

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
				return fmt.Errorf("failed to get source alert: %w", err)
			}

			if len(sourceAlert.Embedding) == 0 {
				return fmt.Errorf("source alert does not have an embedding vector")
			}

			// Convert embedding to float64 for search
			embedding := make([]float64, len(sourceAlert.Embedding))
			for i, v := range sourceAlert.Embedding {
				embedding[i] = float64(v)
			}

			// Search for similar alerts (use large limit for Firestore, filter later)
			similarAlerts, err := repo.SearchSimilarAlerts(ctx, embedding, 100)
			if err != nil {
				return fmt.Errorf("failed to search similar alerts: %w", err)
			}

			// Filter and calculate cosine distances
			type alertWithDistance struct {
				alert    *model.Alert
				distance float64
			}
			var filtered []alertWithDistance

			for _, alert := range similarAlerts {
				// Skip the source alert itself
				if alert.ID == sourceAlert.ID {
					continue
				}

				// Calculate cosine distance
				distance := cosineDistance(sourceAlert.Embedding, alert.Embedding)

				// Apply threshold filter
				if distance > threshold {
					continue
				}

				// Apply keyword filters (AND condition)
				if len(filters) > 0 {
					titleLower := strings.ToLower(alert.Title)
					descLower := strings.ToLower(alert.Description)
					allMatch := true
					for _, filter := range filters {
						filterLower := strings.ToLower(filter)
						if !strings.Contains(titleLower, filterLower) && !strings.Contains(descLower, filterLower) {
							allMatch = false
							break
						}
					}
					if !allMatch {
						continue
					}
				}

				filtered = append(filtered, alertWithDistance{
					alert:    alert,
					distance: distance,
				})
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
			for i, item := range filtered {
				fmt.Fprintf(c.Root().Writer, "%d. %s (distance: %.4f)\n", i+1, item.alert.ID, item.distance)
				fmt.Fprintf(c.Root().Writer, "   Title: %s\n", item.alert.Title)
				if item.alert.Description != "" {
					fmt.Fprintf(c.Root().Writer, "   Description: %s\n", item.alert.Description)
				}
				fmt.Fprintf(c.Root().Writer, "\n")
			}

			return nil
		},
	}
}

// cosineDistance calculates cosine distance between two vectors
func cosineDistance(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 2.0 // Maximum distance
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 2.0
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	return 1.0 - similarity
}
