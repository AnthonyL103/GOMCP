// parse/parse_server_protocol/parse_server.go
package parseserverprotocol

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
	
	"gomcp/server"
	"gomcp/tool"
)

func ParseServerConfig(filePath string) (*server.MCPServer, *RuntimeConfig, error) {
	// Read and unmarshal YAML
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read server config at %s: %w", filePath, err)
	}
	
	var config ServerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, nil, fmt.Errorf("failed to parse YAML at %s: %w", filePath, err)
	}
	
	// Validate server-level config
	if err := validateServerConfig(&config); err != nil {
		return nil, nil, fmt.Errorf("invalid server config at %s: %w", filePath, err)
	}

	handlerSet := make(map[string]string)
	
	// Create tools using the same validation logic
	tools := make([]*tool.Tool, 0, len(config.Tools))
	for _, tc := range config.Tools {

		// check for duplicate handlers
		// two values need to be extracted from maps in go we dont care about the value key just the existence
		// in go maps actually return two values by default which is the value and the boolean existence check
		if _, exists := handlerSet[tc.Handler]; exists {
			return nil, nil, fmt.Errorf("duplicate handler '%s' found in tools at %s", tc.Handler, filePath)
		}
		handlerSet[tc.Handler] = tc.ToolID

		// Convert properties
		props := make(map[string]tool.PropertySchema)
		for name, prop := range tc.InputSchema.Properties {
			props[name] = tool.PropertySchema{
				Type:        prop.Type,
				Description: prop.Description,
			}
		}
		
		schema := tool.JSONSchema{
			Properties: props,
			Required:   tc.InputSchema.Required,
		}
		
		// Reuse the validation function - returns error instead of panicking
		cleanToolID, cleanDesc, cleanHandler, cleanSchema, err := tool.ValidateToolConfig(
			tc.ToolID, 
			tc.Description, 
			"", // handler - we'll set this from tc after
			schema,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid tool '%s': %w", tc.ToolID, err)
		}
		
		// Create tool directly with validated values
		t := &tool.Tool{
			ToolID:      cleanToolID,
			Description: cleanDesc,
			InputSchema: cleanSchema,
			Handler:     tc.Handler, // Set from config
		}
		
		tools = append(tools, t)
	}
	
	mcpServer := server.NewMCPServer(config.ServerID, config.Description, tools)
	return mcpServer, &config.Runtime, nil
}