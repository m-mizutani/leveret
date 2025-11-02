package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// greetParams defines the parameters for the greet tool
type greetParams struct {
	Name string `json:"name" jsonschema:"Name to greet"`
}

// greet implements a simple greeting tool
func greet(ctx context.Context, req *mcp.CallToolRequest, params *greetParams) (*mcp.CallToolResult, any, error) {
	name := params.Name
	if name == "" {
		name = "World"
	}

	response := "Hello, " + name + "!"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
		},
	}, nil, nil
}

func main() {
	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-stdio-server",
		Version: "1.0.0",
	}, nil)

	// Add greet tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "greet",
		Description: "Greet someone by name",
	}, greet)

	// Run server with stdio transport
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
		os.Exit(1)
	}
}
