package adapter_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"google.golang.org/genai"
)

func TestGenerateContent(t *testing.T) {
	projectID := os.Getenv("TEST_GEMINI_PROJECT")
	if projectID == "" {
		t.Skip("TEST_GEMINI_PROJECT is not set")
	}

	ctx := context.Background()
	client, err := adapter.NewGemini(ctx, projectID, "us-central1")
	gt.NoError(t, err)

	contents := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{Text: "Hello, what is the capital of France?"},
			},
		},
	}

	resp, err := client.GenerateContent(ctx, contents, nil)
	if err != nil {
		t.Fatal("failed to call GenerateContent", err)
	}

	if resp == nil ||
		len(resp.Candidates) == 0 ||
		resp.Candidates[0].Content == nil ||
		len(resp.Candidates[0].Content.Parts) == 0 ||
		resp.Candidates[0].Content.Parts[0].Text == "" {
		t.Fatal("unexpected response")
	}

	t.Log("response:", resp.Candidates[0].Content.Parts[0].Text)
}
