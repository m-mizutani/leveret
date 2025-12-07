package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/urfave/cli/v3"
)

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// cosineDistance calculates cosine distance (1 - cosine similarity)
func cosineDistance(a, b []float32) float64 {
	return 1 - cosineSimilarity(a, b)
}

func main() {
	var (
		dimensions     int64
		geminiProject  string
		geminiLocation string
		showVector     bool
		outputJSON     bool
	)

	cmd := &cli.Command{
		Name:  "embedding",
		Usage: "Generate embeddings for given texts",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "dimensions",
				Aliases:     []string{"d"},
				Usage:       "Embedding dimensions",
				Value:       8,
				Sources:     cli.EnvVars("LEVERET_EMBEDDING_DIMENSIONS"),
				Destination: &dimensions,
			},
			&cli.StringFlag{
				Name:        "gemini-project",
				Usage:       "Google Cloud project ID for Gemini API",
				Sources:     cli.EnvVars("LEVERET_GEMINI_PROJECT"),
				Destination: &geminiProject,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "gemini-location",
				Usage:       "Google Cloud location for Gemini API",
				Value:       "us-central1",
				Sources:     cli.EnvVars("LEVERET_GEMINI_LOCATION"),
				Destination: &geminiLocation,
			},
			&cli.BoolFlag{
				Name:        "show-vector",
				Aliases:     []string{"v"},
				Usage:       "Show embedding vectors",
				Sources:     cli.EnvVars("LEVERET_SHOW_VECTOR"),
				Destination: &showVector,
			},
			&cli.BoolFlag{
				Name:        "json",
				Aliases:     []string{"j"},
				Usage:       "Output in JSON format",
				Sources:     cli.EnvVars("LEVERET_OUTPUT_JSON"),
				Destination: &outputJSON,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args().Slice()
			if len(args) == 0 {
				return fmt.Errorf("no text provided")
			}

			gemini, err := adapter.NewGemini(ctx, geminiProject, geminiLocation)
			if err != nil {
				return fmt.Errorf("failed to create Gemini client: %w", err)
			}

			// Generate embeddings for all texts
			type embedding struct {
				text   string
				vector []float32
			}
			embeddings := make([]embedding, 0, len(args))
			for _, text := range args {
				vec, err := gemini.Embedding(ctx, text, int(dimensions))
				if err != nil {
					return fmt.Errorf("failed to generate embedding for %q: %w", text, err)
				}
				embeddings = append(embeddings, embedding{text: text, vector: vec})
			}

			// Calculate pairwise cosine distances
			type distance struct {
				Text1    string  `json:"text1"`
				Text2    string  `json:"text2"`
				Distance float64 `json:"distance"`
			}
			distances := make([]distance, 0)
			for i := 0; i < len(embeddings); i++ {
				for j := i + 1; j < len(embeddings); j++ {
					dist := cosineDistance(embeddings[i].vector, embeddings[j].vector)
					distances = append(distances, distance{
						Text1:    embeddings[i].text,
						Text2:    embeddings[j].text,
						Distance: dist,
					})
				}
			}

			// Output results
			if outputJSON {
				output := map[string]any{
					"embeddings": func() []map[string]any {
						result := make([]map[string]any, len(embeddings))
						for i, e := range embeddings {
							item := map[string]any{
								"text": e.text,
							}
							if showVector {
								item["embedding"] = e.vector
							}
							result[i] = item
						}
						return result
					}(),
					"distances": distances,
				}

				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(output); err != nil {
					return fmt.Errorf("failed to encode results: %w", err)
				}
			} else {
				// Human-readable output
				fmt.Println("=== Embeddings ===")
				for _, e := range embeddings {
					fmt.Printf("Text: %s\n", e.text)
					if showVector {
						fmt.Printf("  Vector: %v\n", e.vector)
					}
				}

				if len(distances) > 0 {
					fmt.Println("\n=== Cosine Distances ===")
					for _, d := range distances {
						fmt.Printf("%s <-> %s: %.4f\n", d.Text1, d.Text2, d.Distance)
					}
				}
			}

			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
