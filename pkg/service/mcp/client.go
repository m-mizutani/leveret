package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

// Client manages connections to multiple MCP servers
type Client struct {
	servers map[string]*server
}

type server struct {
	name    string
	client  *mcp.Client
	session *mcp.ClientSession
	tools   []*mcp.Tool
}

// ServerConfig represents configuration for a single MCP server
type ServerConfig struct {
	Name      string
	Transport string // "stdio" or "http"
	Command   []string
	URL       string
	Env       map[string]string
}

// NewClient creates a new MCP client
func NewClient() *Client {
	return &Client{
		servers: make(map[string]*server),
	}
}

// Connect connects to an MCP server with the given configuration
func (c *Client) Connect(ctx context.Context, cfg ServerConfig) error {
	if _, exists := c.servers[cfg.Name]; exists {
		return goerr.New("server already connected", goerr.V("name", cfg.Name))
	}

	// Create MCP client
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "leveret",
		Version: "0.1.0",
	}, nil)

	var transport mcp.Transport
	var err error

	switch cfg.Transport {
	case "stdio":
		transport, err = c.createStdioTransport(cfg)
	case "http":
		transport, err = c.createHTTPTransport(cfg)
	default:
		return goerr.New("unsupported transport",
			goerr.V("transport", cfg.Transport),
			goerr.V("supported", []string{"stdio", "http"}))
	}

	if err != nil {
		return goerr.Wrap(err, "failed to create transport",
			goerr.V("server", cfg.Name))
	}

	// Connect to MCP server
	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return goerr.Wrap(err, "failed to connect to MCP server",
			goerr.V("server", cfg.Name))
	}

	// List available tools
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return goerr.Wrap(err, "failed to list tools",
			goerr.V("server", cfg.Name))
	}

	c.servers[cfg.Name] = &server{
		name:    cfg.Name,
		client:  mcpClient,
		session: session,
		tools:   toolsResult.Tools,
	}

	return nil
}

// createStdioTransport creates a stdio transport for MCP
func (c *Client) createStdioTransport(cfg ServerConfig) (mcp.Transport, error) {
	if len(cfg.Command) == 0 {
		return nil, goerr.New("command is required for stdio transport")
	}

	cmd := exec.Command(cfg.Command[0], cfg.Command[1:]...)

	// Set environment variables
	if len(cfg.Env) > 0 {
		env := cmd.Env
		for k, v := range cfg.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	return &mcp.CommandTransport{Command: cmd}, nil
}

// createHTTPTransport creates an HTTP transport for MCP
func (c *Client) createHTTPTransport(cfg ServerConfig) (mcp.Transport, error) {
	if cfg.URL == "" {
		return nil, goerr.New("url is required for http transport")
	}

	return &mcp.StreamableClientTransport{
		Endpoint: cfg.URL,
	}, nil
}

// GetTools returns all tools from a specific server
func (c *Client) GetTools(serverName string) ([]*mcp.Tool, error) {
	srv, exists := c.servers[serverName]
	if !exists {
		return nil, goerr.New("server not found", goerr.V("name", serverName))
	}
	return srv.tools, nil
}

// GetAllServers returns names of all connected servers
func (c *Client) GetAllServers() []string {
	names := make([]string, 0, len(c.servers))
	for name := range c.servers {
		names = append(names, name)
	}
	return names
}

// CallTool calls a tool on a specific server
func (c *Client) CallTool(ctx context.Context, serverName string, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
	srv, exists := c.servers[serverName]
	if !exists {
		return nil, goerr.New("server not found", goerr.V("name", serverName))
	}

	result, err := srv.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to call tool",
			goerr.V("server", serverName),
			goerr.V("tool", toolName))
	}

	return result, nil
}

// Close closes all MCP server connections
func (c *Client) Close() error {
	for name, srv := range c.servers {
		if err := srv.session.Close(); err != nil {
			return goerr.Wrap(err, "failed to close session",
				goerr.V("server", name))
		}
	}
	c.servers = make(map[string]*server)
	return nil
}

// Config represents the MCP configuration file structure
type Config struct {
	Servers []ServerConfig `yaml:"servers"`
}

// LoadAndConnect loads MCP configuration from file and connects to all servers
// Returns a tool.Tool provider if successful, nil if no config or connection fails
func LoadAndConnect(ctx context.Context, configPath string) (tool.Tool, error) {
	if configPath == "" {
		return nil, nil // MCP config not specified
	}

	// Get absolute path of config file
	absConfigPath, err := getAbsPath(configPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to resolve config path",
			goerr.V("path", configPath))
	}

	// Load config file
	data, err := os.ReadFile(absConfigPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read MCP config file",
			goerr.V("path", absConfigPath))
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, goerr.Wrap(err, "failed to parse MCP config file",
			goerr.V("path", absConfigPath))
	}

	if len(cfg.Servers) == 0 {
		fmt.Println("No MCP servers configured")
		return nil, nil // No servers configured
	}

	// Create client and connect to all servers
	client := NewClient()
	var connectedServers []string
	var failedServers []string

	for _, serverCfg := range cfg.Servers {
		if err := client.Connect(ctx, serverCfg); err != nil {
			fmt.Printf("Warning: failed to connect to MCP server '%s': %v\n", serverCfg.Name, err)
			failedServers = append(failedServers, serverCfg.Name)
			continue
		}
		fmt.Printf("Connected to MCP server: %s\n", serverCfg.Name)
		connectedServers = append(connectedServers, serverCfg.Name)
	}

	// Return provider if any server connected, otherwise just warn
	if len(connectedServers) == 0 {
		fmt.Printf("Warning: no MCP servers connected (%d failed)\n", len(failedServers))
		return nil, nil // Don't fail, just skip MCP
	}

	return NewProvider(client), nil
}

// getAbsPath returns absolute path, resolving relative paths from current directory
func getAbsPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Abs(path)
}
