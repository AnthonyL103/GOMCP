// protocol/llmprotocol/shared.go
package llmprotocol

import (
    "fmt"
    "github.com/AnthonyL103/GOMCP/Agent"
    "encoding/json"
)

// GetAgentInstructions builds the system prompt for the LLM
func GetAgentInstructions(ag *agent.Agent) string {
    details := ag.GetAgentDetails(ag)
    prompt := fmt.Sprintf("You are %s. %s\n\nYou have access to %d tools across %d servers.", 
        details.AgentID, details.Description, details.ToolCount, details.ServerCount)
    
    if ag.ServerGeneration {
        prompt += "\n\nSERVER GENERATION CAPABILITY ENABLED:\n"
        prompt += "You can create custom tool servers with the generate_server_code tool.\n"
        prompt += "- Handler code is Go function body only (not full functions)\n"
        prompt += "- Use r *http.Request to access POST data\n"
        prompt += "- Use w http.ResponseWriter to send responses\n"
        prompt += "- Write JSON responses using json.Marshal/Unmarshal\n"
        prompt += "- Auto-included imports: net/http, encoding/json, log\n"
        prompt += "- Provide additional imports only if needed (e.g., crypto/md5, strings, strconv)\n"
        prompt += "- Servers start at port 9000 and auto-increment\n"
        prompt += "- Max 5 concurrent servers allowed\n"
        prompt += "- Workflow: generate_server_code → deploy_and_test_tools → deploy_and_register_server → cleanup_server_generation"
    }
    
    return prompt
}

type ToolInfo struct {
    ServerID    string                 // Add this!
    Description string
    Schema      map[string]interface{}
    Handler string
}

func ExtractTools(ag *agent.Agent) map[string]ToolInfo {
    tools := make(map[string]ToolInfo)
    
    for serverID, server := range ag.Registry.Servers {
        for _, tool := range server.Tools {
            
            schemaBytes, _ := json.Marshal(tool.InputSchema)
            var schemaMap map[string]interface{}
            json.Unmarshal(schemaBytes, &schemaMap)
            
            tools[tool.ToolID] = ToolInfo{
                ServerID:    serverID,  
                Description: tool.Description,
                Schema:      schemaMap,
                Handler:     tool.Handler,
            }
        }
    }
    
    return tools
}
