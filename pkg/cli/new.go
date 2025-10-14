package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/usecase/alert"
	"github.com/urfave/cli/v3"
)

func newCommand() *cli.Command {
	var (
		cfg       config
		inputPath string
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "input",
			Aliases:     []string{"i"},
			Usage:       "Path to JSON file containing alert data",
			Sources:     cli.EnvVars("LEVERET_INPUT"),
			Destination: &inputPath,
		},
	}
	flags = append(flags, globalFlags(&cfg)...)
	flags = append(flags, llmFlags(&cfg)...)

	return &cli.Command{
		Name:  "new",
		Usage: "Create a new alert from JSON input",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			if inputPath == "" {
				return goerr.New("input file path is required")
			}

			// Read JSON file
			data, err := os.ReadFile(inputPath)
			if err != nil {
				return goerr.Wrap(err, "failed to read input file", goerr.Value("path", inputPath))
			}

			var alertData any
			if err := json.Unmarshal(data, &alertData); err != nil {
				return goerr.Wrap(err, "failed to parse JSON")
			}

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

			// Insert alert
			newAlert, err := uc.Insert(ctx, alertData)
			if err != nil {
				return goerr.Wrap(err, "failed to insert alert")
			}

			fmt.Fprintf(c.Root().Writer, "Alert created: %s\n", newAlert.ID)
			return nil
		},
	}
}
