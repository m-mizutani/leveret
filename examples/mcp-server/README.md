# Log Search MCP Server Example

This directory contains a simple MCP (Model Context Protocol) server implementation that searches log files.

## Files

- `main.go`: MCP server implementation with `search_logs` tool
- `config.yaml`: MCP configuration file for this server
- `app.log`: Sample application log file
- `security.log`: Sample security log file
- `go.mod`, `go.sum`: Go module files

## Features

The server provides one tool:

### `search_logs`

- **Description**: Search for keyword in log files (*.log) in the current directory
- **Parameters**:
  - `keyword` (string, required): Keyword to search in log files (case-insensitive)
- **Returns**: Matching log lines with filename and line number

### Example

Input:
```json
{
  "keyword": "error"
}
```

Output:
```
Found 3 matches:
app.log:4: 2024-01-15 10:25:33 ERROR Failed to connect to external API: timeout
app.log:9: 2024-01-15 10:32:10 ERROR Authentication failed for user: admin@example.com
app.log:13: 2024-01-15 10:50:22 ERROR Database query timeout: SELECT * FROM large_table
```

## Usage

### 1. Run leveret with the MCP server

From this directory:

```bash
cd examples/mcp-server
leveret chat --mcp-config config.yaml --alert-id <your-alert-id>
```

Or from the repository root:

```bash
leveret chat --mcp-config examples/mcp-server/config.yaml --alert-id <your-alert-id>
```

### 2. Use the log search tool in chat

Once in the chat session, you can ask questions that will trigger the log search tool:

```
> Search for errors in the logs
> Find all authentication failures
> Look for suspicious activity
> Show me database-related issues
```

The server will search all `*.log` files in the `examples/mcp-server/` directory.

## Testing the server directly

You can test the server standalone:

```bash
cd examples/mcp-server
go run main.go
```

The server will start in stdio mode and wait for MCP protocol messages on stdin.

## Extending

You can modify `main.go` to add more tools or customize the search behavior:

```go
// Add another tool
mcp.AddTool(server, &mcp.Tool{
    Name:        "count_logs",
    Description: "Count total log entries",
}, countLogs)
```

Or customize the search logic:

```go
// Add filtering by log level
if params.Level != "" {
    if !strings.Contains(line, params.Level) {
        continue
    }
}
```
