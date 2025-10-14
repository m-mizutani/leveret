package cli

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/usecase/alert"
	"github.com/urfave/cli/v3"
)

func resolveCommand() *cli.Command {
	var (
		cfg        config
		conclusion string
		note       string
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "conclusion",
			Aliases:     []string{"c"},
			Usage:       "Conclusion (unaffected, false_positive, true_positive, inconclusive)",
			Value:       string(model.ConclusionUnaffected),
			Sources:     cli.EnvVars("LEVERET_RESOLVE_CONCLUSION"),
			Destination: &conclusion,
		},
		&cli.StringFlag{
			Name:        "note",
			Aliases:     []string{"n"},
			Usage:       "Additional note",
			Sources:     cli.EnvVars("LEVERET_RESOLVE_NOTE"),
			Destination: &note,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)

	return &cli.Command{
		Name:      "resolve",
		Usage:     "Mark an alert as resolved",
		ArgsUsage: "<alert-id>",
		Flags:     flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() == 0 {
				return goerr.New("alert-id is required")
			}
			alertID := model.AlertID(c.Args().Get(0))

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
			uc := alert.New(repo, claude, gemini)

			// Resolve alert
			if err := uc.Resolve(ctx, alertID, model.Conclusion(conclusion), note); err != nil {
				return goerr.Wrap(err, "failed to resolve alert")
			}

			fmt.Fprintf(c.Root().Writer, "Alert resolved: %s\n", alertID)
			return nil
		},
	}
}
