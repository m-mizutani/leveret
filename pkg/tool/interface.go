package tool

import (
	"context"

	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

// Tool represents an external tool that can be called by the LLM
type Tool interface {
	// Spec returns the tool specification for Gemini function calling
	Spec() *genai.Tool

	// Execute runs the tool with the given function call and returns the response
	Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error)

	// Prompt returns additional information to be added to the system prompt
	// Returns empty string if no additional prompt is needed
	Prompt(ctx context.Context) string

	// Flags returns CLI flags for this tool
	// Returns nil if no flags are needed
	Flags() []cli.Flag
}
