package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"google.golang.org/genai"
)

func main() {
	if err := Run(context.Background()); err != nil {
		log.Fatalf("Error: %+v", err)
	}
}

func Run(ctx context.Context) error {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  os.Getenv("GEMINI_PROJECT"),
		Location: "us-central1",
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return err
	}

	largeText := strings.Repeat("This is a test. ", 500_000) // ~8M characters

	resp, err := client.Models.GenerateContent(ctx,
		"gemini-2.5-flash",
		genai.Text(largeText+"\n\nSummarize the above."),
		&genai.GenerateContentConfig{Temperature: genai.Ptr(float32(0.7))},
	)

	if err != nil {
		var apiErr genai.APIError
		if errors.As(err, &apiErr) {
			fmt.Printf("APIError - Code:%d Status:%s Message:%s Details:%+v\n",
				apiErr.Code, apiErr.Status, apiErr.Message, apiErr.Details)
		}
		return err
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		fmt.Printf("Response: %v\n", resp.Candidates[0].Content.Parts[0])
	}
	return nil
}
