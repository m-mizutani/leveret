package tool

import (
	"context"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
	"google.golang.org/genai"
)

var errToolNotFound = goerr.New("tool not found")

// Registry manages available tools for the LLM
type Registry struct {
	tools     map[string]Tool
	allTools  []Tool
	toolSpecs map[*genai.Tool]bool
}

// New creates a new tool registry with the given tools
// Tools are not registered until Init() is called
func New(tools ...Tool) *Registry {
	return &Registry{
		tools:     make(map[string]Tool),
		allTools:  tools,
		toolSpecs: make(map[*genai.Tool]bool),
	}
}

// Specs returns all tool specifications for Gemini function calling
// All function declarations are combined into a single Tool to avoid
// Gemini API error: "Multiple tools are supported only when they are all search tools"
func (r *Registry) Specs() []*genai.Tool {
	if len(r.toolSpecs) == 0 {
		return nil
	}

	// Combine all function declarations into a single Tool
	var allDeclarations []*genai.FunctionDeclaration
	for spec := range r.toolSpecs {
		if spec.FunctionDeclarations != nil {
			allDeclarations = append(allDeclarations, spec.FunctionDeclarations...)
		}
	}

	if len(allDeclarations) == 0 {
		return nil
	}

	return []*genai.Tool{
		{
			FunctionDeclarations: allDeclarations,
		},
	}
}

// Prompts returns all tool prompts concatenated
func (r *Registry) Prompts(ctx context.Context) string {
	var prompts []string
	for _, t := range r.allTools {
		if prompt := t.Prompt(ctx); prompt != "" {
			prompts = append(prompts, prompt)
		}
	}
	return strings.Join(prompts, "\n\n")
}

// Flags returns all tool flags combined
func (r *Registry) Flags() []cli.Flag {
	var flags []cli.Flag
	for _, t := range r.allTools {
		if toolFlags := t.Flags(); toolFlags != nil {
			flags = append(flags, toolFlags...)
		}
	}
	return flags
}

// AddTool adds a tool to the registry dynamically
// This should be called before Init()
func (r *Registry) AddTool(t Tool) {
	r.allTools = append(r.allTools, t)
}

// Init initializes all tools and registers enabled tools
func (r *Registry) Init(ctx context.Context, client *Client) error {
	for _, t := range r.allTools {
		// Initialize tool and check if enabled
		enabled, err := t.Init(ctx, client)
		if err != nil {
			return goerr.Wrap(err, "failed to initialize tool")
		}

		// Skip if not enabled
		if !enabled {
			continue
		}

		// Register enabled tool
		spec := t.Spec()
		if spec == nil || len(spec.FunctionDeclarations) == 0 {
			continue
		}

		// Register tool spec
		r.toolSpecs[spec] = true

		// Register function declarations with duplicate check
		for _, fd := range spec.FunctionDeclarations {
			if existing, exists := r.tools[fd.Name]; exists {
				// Check if it's the same tool (same pointer)
				if existing != t {
					return goerr.New("duplicate function name", goerr.V("name", fd.Name))
				}
				// Same tool, skip registration
				continue
			}
			r.tools[fd.Name] = t
		}
	}

	return nil
}

// EnabledTools returns the list of enabled tool names
func (r *Registry) EnabledTools() []string {
	tools := make([]string, 0, len(r.tools))
	for name := range r.tools {
		tools = append(tools, name)
	}
	return tools
}

// Tools returns all enabled tools
func (r *Registry) Tools() []Tool {
	// Return unique tools (dedup by pointer)
	seen := make(map[Tool]bool)
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}

// Execute runs the tool with the given function call
func (r *Registry) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	tool, ok := r.tools[fc.Name]
	if !ok {
		return nil, goerr.Wrap(errToolNotFound, "tool not found", goerr.V("name", fc.Name))
	}

	return tool.Execute(ctx, fc)
}
