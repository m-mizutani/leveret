package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
	"google.golang.org/genai"
)

type searchAlertsInput struct {
	Field     string `json:"field"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
	ValueType string `json:"value_type"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
}

type SearchAlerts struct {
	repo repository.Repository
}

// NewSearchAlerts creates a new search_alerts tool
func NewSearchAlerts(repo repository.Repository) *SearchAlerts {
	return &SearchAlerts{
		repo: repo,
	}
}

// FunctionDeclaration returns the function declaration for Gemini API
func (s *SearchAlerts) FunctionDeclaration() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:        "search_alerts",
		Description: `Search alerts by querying fields in the original alert data. Field paths are automatically prefixed with "Data."`,
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"field": {
					Type:        genai.TypeString,
					Description: `Field path in alert data (auto-prefixed with "Data."). Use dot notation for nested fields. The field path must exactly match the structure in the Data field of the alert. Examples: "Type", "Severity", "Service.Action.ActionType", "Resource.InstanceDetails.InstanceId"`,
				},
				"operator": {
					Type:        genai.TypeString,
					Description: "Firestore comparison operator",
					Enum:        []string{"==", "!=", "<", "<=", ">", ">=", "array-contains", "array-contains-any", "in", "not-in"},
				},
				"value": {
					Type:        genai.TypeString,
					Description: "Value to compare",
				},
				"value_type": {
					Type:        genai.TypeString,
					Description: "Type of the value (default: string)",
					Enum:        []string{"string", "number", "boolean", "array"},
				},
				"limit": {
					Type:        genai.TypeInteger,
					Description: "Max results (default: 10, max: 100)",
				},
				"offset": {
					Type:        genai.TypeInteger,
					Description: "Skip count for pagination (default: 0)",
				},
			},
			Required: []string{"field", "operator", "value"},
		},
	}
}

// Execute runs the tool with given parameters
func (s *SearchAlerts) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var input searchAlertsInput
	if err := json.Unmarshal(params, &input); err != nil {
		return "", goerr.Wrap(err, "failed to parse input parameters")
	}

	// Set default value_type to string if not specified
	if input.ValueType == "" {
		input.ValueType = "string"
	}

	// Convert value based on value_type
	convertedValue, err := convertValueByType(input.Value, input.ValueType)
	if err != nil {
		return "", goerr.Wrap(err, "failed to convert value")
	}

	// Search via repository
	alerts, err := s.repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
		Field:    input.Field,
		Operator: input.Operator,
		Value:    convertedValue,
		Limit:    input.Limit,
		Offset:   input.Offset,
	})
	if err != nil {
		return "", goerr.Wrap(err, "failed to search alerts")
	}

	return formatResult(alerts), nil
}

// convertValueByType converts string value to the specified type
func convertValueByType(value string, valueType string) (any, error) {
	switch valueType {
	case "string":
		return value, nil

	case "number":
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse number")
		}
		return num, nil

	case "boolean":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse boolean")
		}
		return b, nil

	case "array":
		// Parse JSON array
		var arr []any
		if err := json.Unmarshal([]byte(value), &arr); err != nil {
			return nil, goerr.Wrap(err, "failed to parse array")
		}
		return arr, nil

	default:
		return nil, goerr.New("unsupported value_type", goerr.Value("type", valueType))
	}
}

// formatResult formats the search result as a human-readable string
func formatResult(alerts []*model.Alert) string {
	if len(alerts) == 0 {
		return "No alerts found matching the criteria."
	}

	result := fmt.Sprintf("Found %d alert(s):\n\n", len(alerts))
	for i, alert := range alerts {
		result += fmt.Sprintf("%d. ID: %s\n", i+1, alert.ID)
		result += fmt.Sprintf("   Title: %s\n", alert.Title)
		result += fmt.Sprintf("   Created: %s\n", alert.CreatedAt.Format("2006-01-02 15:04:05"))
		if alert.Description != "" {
			result += fmt.Sprintf("   Description: %s\n", alert.Description)
		}
		result += "\n"
	}

	return result
}
