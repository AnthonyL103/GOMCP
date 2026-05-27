package llmprotocol

import (
	"encoding/json"
	"fmt"

	agent "github.com/AnthonyL103/GOMCP/Agent"
	"github.com/AnthonyL103/GOMCP/infrageneration"
)

// GetAgentInstructions builds the system prompt for the LLM
func GetAgentInstructions(ag *agent.Agent) string {
	details := ag.GetAgentDetails(ag)
	prompt := fmt.Sprintf("You are %s. %s\n\nYou have access to %d tools across %d servers.",
		details.AgentID, details.Description, details.ToolCount, details.ServerCount)
	prompt += "\n\nRESPONSE FORMAT:\n"
	prompt += "- Respond in clear markdown so the chat UI can render your message cleanly.\n"
	prompt += "- Use fenced code blocks for Terraform and any other code snippets.\n"
	prompt += "- Use headings, bullets, and short paragraphs when they improve readability.\n"
	prompt += "- Do not wrap normal responses in custom XML-like tags or special block markers.\n"

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

	if ag.InfraGeneration {
		prompt += "\n\nINFRA GENERATION CAPABILITY ENABLED:\n"
		prompt += "You can run the AWS preview workflow with the infrageneration tools.\n"
		prompt += "- Workflow: collect_aws_requirements → collect_aws_credentials → generate_aws_terraform_iteration → validate_aws_terraform_iteration → deploy_aws_terraform_iteration → save_project_session\n"
		prompt += "- After a successful deployment, ask the user if they want to save the project session. If they agree, call save_project_session using their email as user_id.\n"
		prompt += "- The backend already stores the chat history from the live conversation, so the save call only needs the user id, project id, project name, and final Terraform script.\n"
		prompt += "- The deploy stage is currently a preview stub and returns true without making cloud changes\n"
		prompt += fmt.Sprintf("- Available tools: %s, %s, %s, %s, %s\n",
			infrageneration.ToolCollectAWSRequirements,
			infrageneration.ToolCollectAWSCredentials,
			infrageneration.ToolGenerateAWSTerraform,
			infrageneration.ToolValidateAWSTerraform,
			infrageneration.ToolDeployAWSTerraform,
		)
		prompt += fmt.Sprintf("- Save tool: %s\n", infrageneration.ToolSaveProjectSession)
	}

	return prompt
}

type ToolInfo struct {
	ServerID    string // Add this!
	Description string
	Schema      map[string]interface{}
	Handler     string
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
