package mcp

import (
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/genai"
)

// convertJSONSchemaToGenai converts JSON Schema to Gemini genai.Schema
func convertJSONSchemaToGenai(schema *jsonschema.Schema) (*genai.Schema, error) {
	if schema == nil {
		return nil, nil
	}

	genaiSchema := &genai.Schema{}

	// Map type
	switch schema.Type {
	case "object":
		genaiSchema.Type = genai.TypeObject
	case "string":
		genaiSchema.Type = genai.TypeString
	case "number", "integer":
		genaiSchema.Type = genai.TypeNumber
	case "boolean":
		genaiSchema.Type = genai.TypeBoolean
	case "array":
		genaiSchema.Type = genai.TypeArray
	default:
		if schema.Type != "" {
			return nil, goerr.New("unsupported schema type", goerr.V("type", schema.Type))
		}
	}

	// Map description
	if schema.Description != "" {
		genaiSchema.Description = schema.Description
	}

	// Map enum values
	if len(schema.Enum) > 0 {
		genaiSchema.Enum = make([]string, len(schema.Enum))
		for i, v := range schema.Enum {
			if s, ok := v.(string); ok {
				genaiSchema.Enum[i] = s
			}
		}
	}

	// Map properties for object type
	if len(schema.Properties) > 0 {
		genaiSchema.Properties = make(map[string]*genai.Schema)
		for name, propSchema := range schema.Properties {
			converted, err := convertJSONSchemaToGenai(propSchema)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to convert property schema",
					goerr.V("property", name))
			}
			genaiSchema.Properties[name] = converted
		}
	}

	// Map required fields
	if len(schema.Required) > 0 {
		genaiSchema.Required = schema.Required
	}

	// Map items for array type
	if schema.Items != nil {
		converted, err := convertJSONSchemaToGenai(schema.Items)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to convert items schema")
		}
		genaiSchema.Items = converted
	}

	return genaiSchema, nil
}
