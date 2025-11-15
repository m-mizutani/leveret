package chat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/usecase/chat"
	"google.golang.org/genai"
)

// mockGemini is a mock implementation of adapter.Gemini for testing
type mockGemini struct {
	generateFunc func(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
	embeddingFunc func(ctx context.Context, text string) (*genai.EmbedContentResponse, error)
}

func (m *mockGemini) GenerateContent(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, contents, config)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGemini) CreateChat(ctx context.Context, config *genai.GenerateContentConfig, history []*genai.Content) (*genai.Chat, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGemini) Embedding(ctx context.Context, text string) (*genai.EmbedContentResponse, error) {
	if m.embeddingFunc != nil {
		return m.embeddingFunc(ctx, text)
	}
	return nil, errors.New("not implemented")
}

func TestIsTokenLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "actual Gemini token limit error",
			err: genai.APIError{
				Code:    400,
				Status:  "INVALID_ARGUMENT",
				Message: "The input token count (2500030) exceeds the maximum number of tokens allowed (1048576).",
			},
			expected: true,
		},
		{
			name: "400 INVALID_ARGUMENT but unrelated",
			err: genai.APIError{
				Code:    400,
				Status:  "INVALID_ARGUMENT",
				Message: "invalid parameter format",
			},
			expected: false,
		},
		{
			name: "500 error",
			err: genai.APIError{
				Code:    500,
				Status:  "INTERNAL_ERROR",
				Message: "internal server error",
			},
			expected: false,
		},
		{
			name:     "other error type",
			err:      errors.New("network timeout"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chat.IsTokenLimitErrorForTest(tt.err)
			gt.V(t, result).Equal(tt.expected)
		})
	}
}

func TestContentSize(t *testing.T) {
	// Test contentSize function indirectly through compression
	content := &genai.Content{
		Role: genai.RoleUser,
		Parts: []*genai.Part{
			{Text: "Hello, this is a test message"},
		},
	}

	// contentSize is private, so we verify it indirectly through compressHistory
	// The function should return non-zero for valid content
	_ = content
}

func TestCompressHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("empty history", func(t *testing.T) {
		mock := &mockGemini{}
		contents := []*genai.Content{}

		_, err := chat.CompressHistoryForTest(ctx, mock, contents)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("history is empty")
	})

	t.Run("successful compression", func(t *testing.T) {
		// Create history with multiple messages
		contents := []*genai.Content{
			genai.NewContentFromText("First user message", genai.RoleUser),
			genai.NewContentFromText("First model response", genai.RoleModel),
			genai.NewContentFromText("Second user message", genai.RoleUser),
			genai.NewContentFromText("Second model response", genai.RoleModel),
			genai.NewContentFromText("Third user message", genai.RoleUser),
			genai.NewContentFromText("Third model response", genai.RoleModel),
		}

		initialCount := len(contents)

		mock := &mockGemini{
			generateFunc: func(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
				// Return mock summary
				return &genai.GenerateContentResponse{
					Candidates: []*genai.Candidate{
						{
							Content: &genai.Content{
								Parts: []*genai.Part{
									{Text: "**Key Findings**: Test summary\n**IOCs Identified**: None\n**Tools Used**: None\n**Decisions**: None"},
								},
							},
						},
					},
				}, nil
			},
		}

		compressed, err := chat.CompressHistoryForTest(ctx, mock, contents)
		gt.NoError(t, err)

		// Verify history was compressed
		if len(compressed) >= initialCount {
			t.Errorf("expected compressed history to be smaller, got %d >= %d", len(compressed), initialCount)
		}

		// Verify first message is the summary
		gt.V(t, compressed[0].Role).Equal(genai.RoleUser)
		gt.V(t, len(compressed[0].Parts)).Equal(1)
		gt.S(t, compressed[0].Parts[0].Text).Contains("Previous Conversation Summary")

		// Verify original contents unchanged
		gt.V(t, len(contents)).Equal(initialCount)
	})

	t.Run("compression failure - summary error", func(t *testing.T) {
		// Create longer history to ensure it passes the 70% threshold
		contents := []*genai.Content{
			genai.NewContentFromText("First message with enough content to make the byte size significant", genai.RoleUser),
			genai.NewContentFromText("Second message with enough content to make the byte size significant", genai.RoleModel),
			genai.NewContentFromText("Third message with enough content to make the byte size significant", genai.RoleUser),
			genai.NewContentFromText("Fourth message with enough content to make the byte size significant", genai.RoleModel),
			genai.NewContentFromText("Fifth message with enough content to make the byte size significant", genai.RoleUser),
		}

		mock := &mockGemini{
			generateFunc: func(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
				return nil, errors.New("API error")
			},
		}

		_, err := chat.CompressHistoryForTest(ctx, mock, contents)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to summarize")
	})

	t.Run("insufficient content to compress", func(t *testing.T) {
		// Very short history that doesn't meet 70% threshold
		contents := []*genai.Content{
			genai.NewContentFromText("x", genai.RoleUser),
		}

		mock := &mockGemini{}

		_, err := chat.CompressHistoryForTest(ctx, mock, contents)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("insufficient content")
	})
}

func TestSummarizeContents(t *testing.T) {
	ctx := context.Background()

	t.Run("successful summarization", func(t *testing.T) {
		contents := []*genai.Content{
			genai.NewContentFromText("Analyzing security alert", genai.RoleUser),
			genai.NewContentFromText("This appears to be a false positive", genai.RoleModel),
		}

		mock := &mockGemini{
			generateFunc: func(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
				return &genai.GenerateContentResponse{
					Candidates: []*genai.Candidate{
						{
							Content: &genai.Content{
								Parts: []*genai.Part{
									{Text: "**Key Findings**: False positive detected"},
								},
							},
						},
					},
				}, nil
			},
		}

		summary, err := chat.SummarizeContentsForTest(ctx, mock, contents)
		gt.NoError(t, err)
		gt.S(t, summary).Contains("Key Findings")
	})

	t.Run("API error", func(t *testing.T) {
		contents := []*genai.Content{
			genai.NewContentFromText("Test", genai.RoleUser),
		}

		mock := &mockGemini{
			generateFunc: func(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
				return nil, errors.New("network error")
			},
		}

		_, err := chat.SummarizeContentsForTest(ctx, mock, contents)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to generate summary")
	})

	t.Run("empty response", func(t *testing.T) {
		contents := []*genai.Content{
			genai.NewContentFromText("Test", genai.RoleUser),
		}

		mock := &mockGemini{
			generateFunc: func(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
				return &genai.GenerateContentResponse{
					Candidates: []*genai.Candidate{},
				}, nil
			},
		}

		_, err := chat.SummarizeContentsForTest(ctx, mock, contents)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("no summary generated")
	})
}
