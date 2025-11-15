package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/genai"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %+v", err)
	}
}

func run() error {
	ctx := context.Background()

	// Get text from command line argument
	if len(os.Args) < 2 {
		return goerr.New("usage: count-token <text>")
	}
	text := os.Args[1]

	// Fixed parameters
	location := "us-central1"
	model := "gemini-2.5-flash"

	// Create Gemini client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  os.Getenv("GEMINI_PROJECT"),
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return goerr.Wrap(err, "failed to create genai client")
	}

	// Count tokens
	contents := genai.Text(text)
	resp, err := client.Models.CountTokens(ctx, model, contents, nil)
	if err != nil {
		return goerr.Wrap(err, "failed to count tokens")
	}

	fmt.Printf("Token count: %d tokens\n", resp.TotalTokens)

	return nil
}
