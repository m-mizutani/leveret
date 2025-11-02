package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type searchLogsParams struct {
	Keyword string `json:"keyword" jsonschema:"The keyword to search for in log files (case-insensitive)"`
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "log-search-server",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_logs",
		Description: "Search for keyword in log files (*.log) in the current directory",
	}, searchLogs)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func searchLogs(ctx context.Context, req *mcp.CallToolRequest, params *searchLogsParams) (*mcp.CallToolResult, any, error) {
	if params.Keyword == "" {
		return nil, nil, fmt.Errorf("keyword is required")
	}

	// Search in *.log files in examples/mcp-server/ directory (from repository root)
	files, err := filepath.Glob("examples/mcp-server/*.log")
	if err != nil {
		return nil, nil, err
	}

	var results []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), strings.ToLower(params.Keyword)) {
				results = append(results, fmt.Sprintf("%s:%d: %s", filepath.Base(file), i+1, line))
			}
		}
	}

	var resultText string
	if len(results) == 0 {
		resultText = fmt.Sprintf("No matches found for keyword: %s", params.Keyword)
	} else {
		resultText = fmt.Sprintf("Found %d matches:\n%s", len(results), strings.Join(results, "\n"))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, nil, nil
}
