package chat

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"google.golang.org/genai"
)

const (
	compressionRatio = 0.7 // Compress first 70% by byte size
)

//go:embed prompt/summarize.md
var summarizePromptRaw string

// isTokenLimitError checks if the error is due to token limit exceeded
func isTokenLimitError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr genai.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	// Check exact Gemini API token limit error pattern
	// Example: "The input token count (2500030) exceeds the maximum number of tokens allowed (1048576)."
	return apiErr.Code == 400 &&
		apiErr.Status == "INVALID_ARGUMENT" &&
		strings.HasPrefix(apiErr.Message, "The input token count (") &&
		strings.Contains(apiErr.Message, ") exceeds the maximum number of tokens allowed (")
}

// contentSize calculates the byte size of a content by JSON marshaling
func contentSize(content *genai.Content) int {
	data, err := json.Marshal(content)
	if err != nil {
		return 0
	}
	return len(data)
}

// compressHistory compresses conversation history based on byte size and returns new compressed contents
func compressHistory(ctx context.Context, gemini adapter.Gemini, contents []*genai.Content) ([]*genai.Content, error) {
	if len(contents) == 0 {
		return nil, goerr.New("history is empty")
	}

	// Calculate byte size for each content
	totalBytes := 0
	byteSizes := make([]int, len(contents))
	for i, content := range contents {
		size := contentSize(content)
		byteSizes[i] = size
		totalBytes += size
	}

	// Calculate compression threshold (70% of total bytes)
	compressThreshold := int(float64(totalBytes) * compressionRatio)

	// Find the index where we cross the 70% threshold
	cumulativeBytes := 0
	compressIndex := 0
	for i, size := range byteSizes {
		cumulativeBytes += size
		if cumulativeBytes >= compressThreshold {
			compressIndex = i + 1 // Include this message in compression
			break
		}
	}

	// If compression index is 0 or at the end, nothing to compress
	if compressIndex == 0 || compressIndex >= len(contents) {
		return nil, goerr.New("insufficient content to compress")
	}

	// Extract contents to compress and to keep
	toCompress := contents[:compressIndex]
	toKeep := contents[compressIndex:]

	// Generate summary of compressed contents
	summary, err := summarizeContents(ctx, gemini, toCompress)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to summarize contents")
	}

	// Create summary content as user message
	summaryContent := &genai.Content{
		Role: genai.RoleUser,
		Parts: []*genai.Part{
			{Text: "=== Previous Conversation Summary ===\n\n" + summary},
		},
	}

	// Return new history: summary + kept contents
	newContents := append([]*genai.Content{summaryContent}, toKeep...)
	return newContents, nil
}

// summarizeContents generates a summary of the given conversation contents
func summarizeContents(ctx context.Context, gemini adapter.Gemini, contents []*genai.Content) (string, error) {
	// Append summarization request as the final user message
	contentsWithPrompt := append(contents, genai.NewContentFromText(summarizePromptRaw, genai.RoleUser))

	// Create request with system instruction
	thinkingBudget := int32(0)
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText("You are an assistant for security alert analysis.", ""),
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: false,
			ThinkingBudget:  &thinkingBudget,
		},
	}

	// Pass contents with prompt to Gemini API
	resp, err := gemini.GenerateContent(ctx, contentsWithPrompt, config)
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate summary")
	}

	// Extract text from response
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", goerr.New("no summary generated")
	}

	var summary strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			summary.WriteString(part.Text)
		}
	}

	if summary.Len() == 0 {
		return "", goerr.New("empty summary generated")
	}

	return summary.String(), nil
}

// Test helpers - exported versions of private functions for testing
// These should only be used in tests

// CompressHistoryForTest is a test helper that exposes compressHistory
func CompressHistoryForTest(ctx context.Context, gemini adapter.Gemini, contents []*genai.Content) ([]*genai.Content, error) {
	return compressHistory(ctx, gemini, contents)
}

// SummarizeContentsForTest is a test helper that exposes summarizeContents
func SummarizeContentsForTest(ctx context.Context, gemini adapter.Gemini, contents []*genai.Content) (string, error) {
	return summarizeContents(ctx, gemini, contents)
}

// IsTokenLimitErrorForTest is a test helper that exposes isTokenLimitError
func IsTokenLimitErrorForTest(err error) bool {
	return isTokenLimitError(err)
}
