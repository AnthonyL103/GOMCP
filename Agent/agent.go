package Agent

import (
	"fmt"
	"strings"
)

func isString(val interface{}) bool {
	_, ok := val.(string)
	return ok
}


type Agent struct {
	AgentID string
	Description string
	Registry *Registry
}

func NewAgent(
	agentID string,
	description string,
	registry *Registry,
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

	return &Agent{
		AgentID:     agentID,
		Description: description,
		Registry:    registry,
	}
}

func (a *Agent)GetAgentDetails(agent *Agent) string {
	if a == nil {
		panic("Agent cannot be nil")
	}

	servers := make(map[string]*MCPServer)
	tools := make(map[string]*Tool)
	for _, server := range a.Registry.Servers {
		servers[server.ServerID] = server

		for _, tool := range server.Tools {
			tools[tool.Name] = tool
			
		}
	}
	return fmt.Sprintf("Agent Details:\nAgentID: %s\nDescription: %s\nServers: %v\nTools: %v", a.AgentID, a.Description, servers, tools)
}