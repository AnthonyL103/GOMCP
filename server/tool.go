package tool

import (
	"fmt"
	"strings"
)

type Tool struct {
	ToolID      string
	Description string
	InputSchema JSONSchema
	HandlerFunc func(input map[string]interface{}) (interface{}, error)
}

type JSONSchema struct {
	Properties map[string]PropertySchema
	Required   []string
}

type PropertySchema struct {
	Type        string
	Description string
}

func NewTool(
	toolID string,
	description string,
	inputSchema JSONSchema,
	handlerFunc func(input map[string]interface{}) (interface{}, error),
) *Tool {
	// Strip whitespace
	toolID = strings.TrimSpace(toolID)
	description = strings.TrimSpace(description)

	// Validation
	if handlerFunc == nil {
		panic("Handler function cannot be nil")
	}
	if toolID == "" {
		panic("ToolID cannot be empty")
	}
	if description == "" {
		panic("Description cannot be empty")
	}
	if len(inputSchema.Properties) == 0 {
		panic("InputSchema must have at least one property")
	}

	// Validate required fields exist in properties
	for _, req := range inputSchema.Required {
		req = strings.TrimSpace(req)
		prop, exists := inputSchema.Properties[req]
		if !exists {
			panic(fmt.Sprintf("Required field '%s' not found in properties", req))
		}
		// Validate property has a type
		if strings.TrimSpace(prop.Type) == "" {
			panic(fmt.Sprintf("Property '%s' must have a type", req))
		}
	}

	// Trim all property names and values, creates a map of string keys and propertyschema vals 
	trimmedProperties := make(map[string]PropertySchema)
	for key, prop := range inputSchema.Properties {
		trimmedKey := strings.TrimSpace(key)
		trimmedProperties[trimmedKey] = PropertySchema{
			Type:        strings.TrimSpace(prop.Type),
			Description: strings.TrimSpace(prop.Description),
		}
	}

	// Trim required fields
	trimmedRequired := make([]string, len(inputSchema.Required))
	for i, req := range inputSchema.Required {
		trimmedRequired[i] = strings.TrimSpace(req)
	}

	return &Tool{
		ToolID:      toolID,
		Description: description,
		InputSchema: JSONSchema{
			Properties: trimmedProperties,
			Required:   trimmedRequired,
		},
		HandlerFunc: handlerFunc,
	}
}

// ValidateInput checks if the input matches the schema
func (t *Tool) ValidateInput(input map[string]interface{}) error {
	// Check all required fields are present
	for _, req := range t.InputSchema.Required {
		if _, exists := input[req]; !exists {
			return fmt.Errorf("required field '%s' is missing", req)
		}
	}

	// Validate types of provided fields
	for key, value := range input {
		prop, exists := t.InputSchema.Properties[key]
		if !exists {
			return fmt.Errorf("unknown field '%s'", key)
		}

		// Type checking
		if !validateType(value, prop.Type) {
			return fmt.Errorf("field '%s' must be of type '%s'", key, prop.Type)
		}
	}

	return nil
}

// Helper function to validate types
func validateType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return false
	}
}

// Execute the tool with validation
func (t *Tool) Execute(input map[string]interface{}) (interface{}, error) {
	// Validate input first
	if err := t.ValidateInput(input); err != nil {
		return nil, err
	}

	return t.HandlerFunc(input)
}
