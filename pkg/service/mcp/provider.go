package mcp

import (
	"context"
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

// Provider implements tool.Tool interface for MCP tools
type Provider struct {
	client *Client
	tools  []*mcpTool
}

type mcpTool struct {
	serverName string
	mcpTool    *mcp.Tool
	funcDecl   *genai.FunctionDeclaration
}

// NewProvider creates a new MCP tool provider
func NewProvider(client *Client) *Provider {
	return &Provider{
		client: client,
		tools:  make([]*mcpTool, 0),
	}
}

// Flags returns CLI flags for MCP provider
func (p *Provider) Flags() []cli.Flag {
	return nil // MCP config is loaded separately
}

// Init initializes the MCP provider and registers tools
func (p *Provider) Init(ctx context.Context, client *tool.Client) (bool, error) {
	if p.client == nil {
		return false, nil // MCP client not configured
	}

	// Get all connected servers
	serverNames := p.client.GetAllServers()
	if len(serverNames) == 0 {
		return false, nil // No servers connected
	}

	// Register tools from each server
	for _, serverName := range serverNames {
		tools, err := p.client.GetTools(serverName)
		if err != nil {
			return false, goerr.Wrap(err, "failed to get tools from server",
				goerr.V("server", serverName))
		}

		for _, t := range tools {
			// Convert MCP tool to Gemini function declaration
			funcDecl, err := p.convertToFunctionDeclaration(t)
			if err != nil {
				return false, goerr.Wrap(err, "failed to convert tool",
					goerr.V("server", serverName),
					goerr.V("tool", t.Name))
			}

			p.tools = append(p.tools, &mcpTool{
				serverName: serverName,
				mcpTool:    t,
				funcDecl:   funcDecl,
			})
		}
	}

	return len(p.tools) > 0, nil
}

// convertToFunctionDeclaration converts MCP tool to Gemini FunctionDeclaration
func (p *Provider) convertToFunctionDeclaration(t *mcp.Tool) (*genai.FunctionDeclaration, error) {
	funcDecl := &genai.FunctionDeclaration{
		Name:        t.Name,
		Description: t.Description,
	}

	// Convert input schema if present
	if t.InputSchema != nil {
		// InputSchema is interface{}, so we need to convert it
		// to jsonschema.Schema through JSON marshaling/unmarshaling
		schemaJSON, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal input schema")
		}

		var jsSchema jsonschema.Schema
		if err := json.Unmarshal(schemaJSON, &jsSchema); err != nil {
			return nil, goerr.Wrap(err, "failed to unmarshal input schema")
		}

		schema, err := convertJSONSchemaToGenai(&jsSchema)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to convert input schema")
		}
		funcDecl.Parameters = schema
	}

	return funcDecl, nil
}

// Spec returns the tool specification for Gemini
func (p *Provider) Spec() *genai.Tool {
	if len(p.tools) == 0 {
		return nil
	}

	funcDecls := make([]*genai.FunctionDeclaration, len(p.tools))
	for i, t := range p.tools {
		funcDecls[i] = t.funcDecl
	}

	return &genai.Tool{
		FunctionDeclarations: funcDecls,
	}
}

// Prompt returns additional prompt information
func (p *Provider) Prompt(ctx context.Context) string {
	if len(p.tools) == 0 {
		return ""
	}

	return "You have access to MCP (Model Context Protocol) tools that provide additional capabilities like file system access, database queries, and web searches."
}

// Execute executes an MCP tool
func (p *Provider) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	// Find the tool
	var targetTool *mcpTool
	for _, t := range p.tools {
		if t.funcDecl.Name == fc.Name {
			targetTool = t
			break
		}
	}

	if targetTool == nil {
		return nil, goerr.New("tool not found", goerr.V("name", fc.Name))
	}

	// Convert arguments
	var arguments map[string]any
	if fc.Args != nil {
		// fc.Args is already map[string]any
		arguments = fc.Args
	}

	// Call MCP tool
	result, err := p.client.CallTool(ctx, targetTool.serverName, targetTool.mcpTool.Name, arguments)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to call MCP tool")
	}

	// Convert result to JSON string
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal result")
	}

	return &genai.FunctionResponse{
		Name:     fc.Name,
		Response: map[string]any{"result": string(resultJSON)},
	}, nil
}
