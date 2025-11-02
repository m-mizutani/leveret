package mcp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/service/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStdioTransport(t *testing.T) {
	ctx := context.Background()

	// Create client
	client := mcp.NewClient()

	// Connect to stdio test server
	err := client.Connect(ctx, mcp.ServerConfig{
		Name:      "test-stdio",
		Transport: "stdio",
		Command:   []string{"go", "run", "./testdata/stdio/main.go"},
	})
	gt.NoError(t, err)
	defer client.Close()

	// Verify server is connected
	servers := client.GetAllServers()
	gt.A(t, servers).Length(1)
	gt.Equal(t, servers[0], "test-stdio")

	// Get tools
	tools, err := client.GetTools("test-stdio")
	gt.NoError(t, err)
	gt.A(t, tools).Length(1)
	gt.Equal(t, tools[0].Name, "greet")

	// Call tool
	result, err := client.CallTool(ctx, "test-stdio", "greet", map[string]any{
		"name": "Leveret",
	})
	gt.NoError(t, err)
	gt.V(t, result).NotNil()
	gt.A(t, result.Content).Length(1)

	textContent, ok := result.Content[0].(*mcpsdk.TextContent)
	gt.True(t, ok)
	gt.Equal(t, textContent.Text, "Hello, Leveret!")
}

func TestHTTPStreamableTransport(t *testing.T) {
	ctx := context.Background()

	// Create MCP server
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "test-http-server",
		Version: "1.0.0",
	}, nil)

	// Add test tool
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "echo",
		Description: "Echo back the message",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, params *struct {
		Message string `json:"message" jsonschema:"Message to echo"`
	}) (*mcpsdk.CallToolResult, any, error) {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: params.Message},
			},
		}, nil, nil
	})

	// Create HTTP handler
	handler := mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		return server
	}, nil)

	// Start test HTTP server
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	// Create client and connect
	client := mcp.NewClient()
	err := client.Connect(ctx, mcp.ServerConfig{
		Name:      "test-http",
		Transport: "http",
		URL:       testServer.URL,
	})
	gt.NoError(t, err)
	defer client.Close()

	// Verify server is connected
	servers := client.GetAllServers()
	gt.A(t, servers).Length(1)
	gt.Equal(t, servers[0], "test-http")

	// Get tools
	tools, err := client.GetTools("test-http")
	gt.NoError(t, err)
	gt.A(t, tools).Length(1)
	gt.Equal(t, tools[0].Name, "echo")

	// Call tool
	result, err := client.CallTool(ctx, "test-http", "echo", map[string]any{
		"message": "Hello from HTTP!",
	})
	gt.NoError(t, err)
	gt.V(t, result).NotNil()
	gt.A(t, result.Content).Length(1)

	textContent, ok := result.Content[0].(*mcpsdk.TextContent)
	gt.True(t, ok)
	gt.Equal(t, textContent.Text, "Hello from HTTP!")
}

func TestMultipleServers(t *testing.T) {
	ctx := context.Background()

	client := mcp.NewClient()

	// Connect to stdio server
	err := client.Connect(ctx, mcp.ServerConfig{
		Name:      "stdio-server",
		Transport: "stdio",
		Command:   []string{"go", "run", "./testdata/stdio/main.go"},
	})
	gt.NoError(t, err)

	// Create and start HTTP server
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "http-server",
		Version: "1.0.0",
	}, nil)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "test",
		Description: "Test tool",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, params *struct{}) (*mcpsdk.CallToolResult, any, error) {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: "test"},
			},
		}, nil, nil
	})

	handler := mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		return server
	}, nil)
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	// Connect to HTTP server
	err = client.Connect(ctx, mcp.ServerConfig{
		Name:      "http-server",
		Transport: "http",
		URL:       testServer.URL,
	})
	gt.NoError(t, err)

	// Verify both servers are connected
	servers := client.GetAllServers()
	gt.A(t, servers).Length(2)

	// Check both servers exist
	serverMap := make(map[string]bool)
	for _, s := range servers {
		serverMap[s] = true
	}
	gt.True(t, serverMap["stdio-server"])
	gt.True(t, serverMap["http-server"])

	// Call tool on stdio server
	result1, err := client.CallTool(ctx, "stdio-server", "greet", map[string]any{
		"name": "Test1",
	})
	gt.NoError(t, err)
	textContent1, ok := result1.Content[0].(*mcpsdk.TextContent)
	gt.True(t, ok)
	gt.Equal(t, textContent1.Text, "Hello, Test1!")

	// Call tool on HTTP server
	result2, err := client.CallTool(ctx, "http-server", "test", map[string]any{})
	gt.NoError(t, err)
	textContent2, ok := result2.Content[0].(*mcpsdk.TextContent)
	gt.True(t, ok)
	gt.Equal(t, textContent2.Text, "test")

	// Close client before test server to allow clean shutdown
	client.Close()
}
