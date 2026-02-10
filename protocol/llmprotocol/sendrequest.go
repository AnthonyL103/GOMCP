package llmprotocol

import (
	"fmt"
	"encoding/json"

	"github.com/AnthonyL103/GOMCP/agent"
)

func isString(val interface{}) bool {
	_, ok := val.(string)
	return ok
}

type LLMRequest struct {
	UserMessage       string
	AgentInstructions string
	AvailableTools    map[string]map[string]string
	ModelID           string
	ThreadContext     string
	Temperature       float32
	MaxTokens         int
	ResponseFormat    string
}

// LLMResponse is what the LLM returns - either text or a tool call
type LLMResponse struct {
	ResponseText  string            // Text response from LLM
	ToolCall      *ToolCall         // If LLM wants to call a tool
	StopReason    string            // why LLM stopped (e.g., "tool_use", "end_turn")
}

// ToolCall represents a tool the LLM wants to invoke
type ToolCall struct {
	ToolName   string                 // Name of the tool to call
	Parameters map[string]interface{} // Tool parameters as JSON
}

// BuildLLMRequest constructs an LLMRequest from parsed agent config
func BuildLLMRequest(ag *agent.Agent, userMessage string) (*LLMRequest, error) {
	if ag == nil || ag.Registry == nil {
		return nil, fmt.Errorf("agent and registry cannot be nil")
	}
	if ag.LLMConfig == nil {
		return nil, fmt.Errorf("agent has no llm config")
	}
	if userMessage == ""  || !isString(userMessage) {
		return nil, fmt.Errorf("user message cannot be empty and must be a string")
	}

	// Flatten all tools from all servers into one map
	availableTools := make(map[string]map[string]string)
	for _, mcp := range ag.Registry.Servers {
		for _, tool := range mcp.Tools {
			// Marshal InputSchema to JSON for LLM consumption
			schemaBytes, err := json.Marshal(tool.InputSchema)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal schema for tool %s: %w", tool.ToolID, err)
			}

			availableTools[tool.ToolID] = map[string]string{
				"description": tool.Description,
				"params":      string(schemaBytes),
			}
		}
	}

	// Build agent instructions from agent details
	details := ag.GetAgentDetails(ag)
	agentInstructions := fmt.Sprintf("You are %s. %s. You have access to %d tools across %d servers.",
		details.AgentID, details.Description, details.ToolCount, details.ServerCount)

	req := &LLMRequest{
		UserMessage:       userMessage,
		AgentInstructions: agentInstructions,
		AvailableTools:    availableTools,
		ModelID:           ag.LLMConfig.Model,
		Temperature:       ag.LLMConfig.Temperature,
		MaxTokens:         ag.LLMConfig.MaxTokens,
		ResponseFormat: `Respond in JSON format. If responding with text: {"responseText": "your response", "stopReason": "end_turn"}. If calling a tool: {"toolCall": {"toolName": "tool_id", "parameters": {...}}, "stopReason": "tool_use"}`,
	}

	return req, nil
}

// PrintLLMRequest prints the LLMRequest for debugging
func PrintLLMRequest(req *LLMRequest) {
	if req == nil {
		fmt.Println("LLMRequest is nil")
		return
	}

	fmt.Println("=== LLM Request ===")
	fmt.Printf("Model: %s\n", req.ModelID)
	fmt.Printf("Temperature: %.2f\n", req.Temperature)
	fmt.Printf("Max Tokens: %d\n", req.MaxTokens)
	fmt.Printf("\nAgent Instructions:\n%s\n", req.AgentInstructions)
	fmt.Printf("\nUser Message:\n%s\n", req.UserMessage)
	
	fmt.Printf("\nAvailable Tools (%d):\n", len(req.AvailableTools))
	for toolName, toolInfo := range req.AvailableTools {
		fmt.Printf("  - %s\n", toolName)
		fmt.Printf("    Description: %s\n", toolInfo["description"])
		fmt.Printf("    Parameters: %s\n", toolInfo["params"])
	}

	fmt.Printf("\nResponse Format:\n%s\n", req.ResponseFormat)
	fmt.Println("==================")
}

func 
