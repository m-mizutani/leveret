# MCP Configuration Example

This directory contains an example MCP (Model Context Protocol) configuration using 3rd party MCP servers.

## Files

- `config.yaml`: Example MCP configuration file with 3rd party servers

## Available 3rd Party MCP Servers

The configuration includes examples of popular MCP servers:

### Filesystem Server
- **Package**: `@modelcontextprotocol/server-filesystem`
- **Description**: Read and search files in specified directories
- **Install**: `npm install -g @modelcontextprotocol/server-filesystem`

### GitHub Server (commented out)
- **Package**: `@modelcontextprotocol/server-github`
- **Description**: Search repositories, issues, and pull requests
- **Install**: `npm install -g @modelcontextprotocol/server-github`
- **Requires**: `GITHUB_PERSONAL_ACCESS_TOKEN` environment variable

### Brave Search Server (commented out)
- **Package**: `@modelcontextprotocol/server-brave-search`
- **Description**: Web search capabilities
- **Install**: `npm install -g @modelcontextprotocol/server-brave-search`
- **Requires**: `BRAVE_API_KEY` environment variable

## Usage

### 1. Install required MCP servers

```bash
# Install filesystem server (enabled by default in config)
npm install -g @modelcontextprotocol/server-filesystem

# Optional: Install other servers and uncomment them in config.yaml
npm install -g @modelcontextprotocol/server-github
npm install -g @modelcontextprotocol/server-brave-search
```

### 2. Configure API keys (if needed)

Edit `config.yaml` and add your API keys:

```yaml
env:
  GITHUB_PERSONAL_ACCESS_TOKEN: your-token-here
  BRAVE_API_KEY: your-api-key-here
```

### 3. Run leveret with MCP configuration

From the repository root:

```bash
leveret chat --mcp-config examples/mcp-config/config.yaml --alert-id <your-alert-id>
```

Or set environment variable:

```bash
export LEVERET_MCP_CONFIG=examples/mcp-config/config.yaml
leveret chat --alert-id <your-alert-id>
```

### 4. Use MCP tools in chat

Once in the chat session, you can ask questions that will use the configured MCP servers:

```
> Read the file /tmp/test.log
> Search for files containing "error" in /tmp
> Find GitHub issues related to authentication
> Search the web for CVE-2024-1234
```

## More MCP Servers

Visit [Model Context Protocol Servers](https://github.com/modelcontextprotocol/servers) for more official MCP servers, or search npm/GitHub for community-built servers.
