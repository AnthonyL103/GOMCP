// protocol/llmprotocol/shared.go
package llmprotocol

import (
    "fmt"
    "github.com/AnthonyL103/GOMCP/agent"
)

// GetAgentInstructions builds the system prompt
func GetAgentInstructions(ag *agent.Agent) string {
    details := ag.GetAgentDetails(ag)
    return fmt.Sprintf("You are %s. %s. You have access to %d tools across %d servers.",
        details.AgentID, details.Description, details.ToolCount, details.ServerCount)
}

type ToolInfo struct {
    ServerID    string                 // Add this!
    Description string
    Schema      map[string]interface{}
}

func ExtractTools(ag *agent.Agent) map[string]ToolInfo {
    tools := make(map[string]ToolInfo)
    
    for serverID, server := range ag.Registry.Servers {
        for _, tool := range server.Tools {
            schemaBytes, _ := json.Marshal(tool.InputSchema)
            var schemaMap map[string]interface{}
            json.Unmarshal(schemaBytes, &schemaMap)
            
            tools[tool.ToolID] = ToolInfo{
                ServerID:    serverID,  // Store it here!
                Description: tool.Description,
                Schema:      schemaMap,
            }
        }
    }
    
    return tools
}

func ExecuteTool(ag *agent.Agent, tc *transport.ToolCall) {

}
