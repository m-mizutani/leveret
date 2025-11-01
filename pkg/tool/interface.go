package tool

import (
	"context"

	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

// Tool represents an external tool that can be called by the LLM
type Tool interface {
	// Flags returns CLI flags for this tool
	// Called first to register CLI flags
	// Returns nil if no flags are needed
	Flags() []cli.Flag

	// Init initializes the tool with the given context and client
	// Called after CLI flags are parsed and before the tool is used
	// Returns (enabled, error) where:
	//   - enabled: true if the tool should be registered and available for use
	//   - error: non-nil if initialization fails
	Init(ctx context.Context, client *Client) (bool, error)

	// Spec returns the tool specification for Gemini function calling
	// Called when building the tool list for LLM
	Spec() *genai.Tool

	// Prompt returns additional information to be added to the system prompt
	// Called when constructing the system prompt
	// Returns empty string if no additional prompt is needed
	Prompt(ctx context.Context) string

	// Execute runs the tool with the given function call and returns the response
	// Called when LLM requests to execute this tool
	Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error)
}
