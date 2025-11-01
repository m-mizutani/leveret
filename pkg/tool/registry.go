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
func New(tools ...Tool) *Registry {
	r := &Registry{
		tools:     make(map[string]Tool),
		allTools:  tools,
		toolSpecs: make(map[*genai.Tool]bool),
	}

	for _, t := range tools {
		spec := t.Spec()
		if spec != nil && len(spec.FunctionDeclarations) > 0 {
			r.toolSpecs[spec] = true
			for _, fd := range spec.FunctionDeclarations {
				r.tools[fd.Name] = t
			}
		}
	}

	return r
}

// Specs returns all tool specifications for Gemini function calling
func (r *Registry) Specs() []*genai.Tool {
	specs := make([]*genai.Tool, 0, len(r.toolSpecs))
	for spec := range r.toolSpecs {
		specs = append(specs, spec)
	}
	return specs
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

// Execute runs the tool with the given function call
func (r *Registry) Execute(ctx context.Context, fc genai.FunctionCall) (*genai.FunctionResponse, error) {
	tool, ok := r.tools[fc.Name]
	if !ok {
		return nil, goerr.Wrap(errToolNotFound, "tool not found", goerr.V("name", fc.Name))
	}

	return tool.Execute(ctx, fc)
}
