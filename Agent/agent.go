package Agent

import (
	"fmt"
	"strings"
)

func isString(val interface{}) bool {
	_, ok := val.(string)
	return ok
}

type AgentDetails struct {
	AgentID string
	Description string
	ServerCount int
	ToolCount int
	// ServerTools maps ServerID -> map of ToolName -> Tool
	ServerTools map[string]map[string]*Tool
}

type LLMConfig struct {
	APIKey string
	Model  string
}


type Agent struct {
	AgentID string
	Description string
	Registry *Registry
	llmConfig *LLMConfig
}

//return list of valid models for the user 
func getModelList() []string {
	return []string{
		"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "o1-preview", "o1-mini",
		"claude-opus-4-5-20251101", "claude-sonnet-4-5-20250929", "claude-haiku-4-5-20251001",
	}
}

func validateLLMConfig(llmConfig *LLMConfig) {
	if llmConfig == nil {
		panic("LLMConfig cannot be nil")
	}

	if llmConfig.APIKey == "" || !isString(llmConfig.APIKey) {
		panic("LLMConfig.APIKey is required and must be a non-empty string")
	}
	if llmConfig.Model == "" || !isString(llmConfig.Model) {
		panic("LLMConfig.Model is required and must be a non-empty string")
	}
	
	validModels := map[string]bool{
		// OpenAI GPT models (current)
		"gpt-4o":              true,
		"gpt-4o-mini":         true,
		"gpt-4-turbo":         true,
		"gpt-4-turbo-preview": true,
		"o1-preview":          true,
		"o1-mini":             true,
		"gpt-3.5-turbo":       true, // legacy but still supported
		
		// Claude 4.5 models (latest)
		"claude-opus-4-5-20251101":     true,
		"claude-sonnet-4-5-20250929":   true,
		"claude-haiku-4-5-20251001":    true,
		
		// Claude 3.5 models (legacy)
		"claude-3-5-sonnet-20241022":   true,
		"claude-3-5-haiku-20241022":    true,
	}
	
	if !validModels[llmConfig.Model] {
		panic(fmt.Sprintf("Invalid model '%s'. Supported models: %v", llmConfig.Model, getModelList()))
	}
}



func NewAgent(
	agentID string,
	description string,
	registry *Registry,
	llmConfig *LLMConfig,
) *Agent {
	agentID = strings.TrimSpace(agentID)
	description = strings.TrimSpace(description)

	if agentID == "" || !isString(agentID) {
		panic("AgentID is required and must be a non-empty string")
	}
	if description == "" || !isString(description) {
		panic("Description is required and must be a non-empty string")
	}

	if registry == nil {
		panic("Registry cannot be nil")
	}

	validateLLMConfig(llmConfig)

	return &Agent{
		AgentID:     agentID,
		Description: description,
		Registry:    registry,
		llmConfig:   llmConfig,
	}
}

func (a *Agent)GetAgentDetails(agent *Agent) *AgentDetails {
	if a == nil {
		panic("Agent cannot be nil")
	}

	servers := make(map[string]*MCPServer)
	serverTools := make(map[string]map[string]*Tool)
	toolCount := 0
	
	for _, server := range a.Registry.Servers {
		servers[server.ServerID] = server
		serverTools[server.ServerID] = make(map[string]*Tool)

		for toolName, tool := range server.Tools {
			serverTools[server.ServerID][toolName] = tool
			toolCount++
		}
	}
	
	return &AgentDetails{
		AgentID:     a.AgentID,
		Description: a.Description,
		ServerCount: len(servers),
		ToolCount:   toolCount,
		ServerTools: serverTools,
	}
}