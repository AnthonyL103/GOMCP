package infrageneration

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agent "github.com/AnthonyL103/GOMCP/Agent"
	"github.com/AnthonyL103/GOMCP/chat"
)

func CollectAWSRequirementsTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	_ = ag
	return formatInfraStagePreview(
		ToolCollectAWSRequirements,
		"AWS requirements collected",
		params,
		"Next: call collect_aws_credentials once the requirements are final.",
	), false
}

func CollectAWSCredentialsTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	_ = ag
	return formatInfraStagePreview(
		ToolCollectAWSCredentials,
		"AWS credential context collected",
		params,
		"Next: call generate_aws_terraform_iteration with the approved requirements and credential summary.",
	), false
}

func GenerateAWSTerraformTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	_ = ag
	return formatInfraStagePreview(
		ToolGenerateAWSTerraform,
		"Terraform draft generated",
		params,
		"Next: call validate_aws_terraform_iteration to review the draft output.",
	), false
}

func ValidateAWSTerraformTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	_ = ag
	return formatInfraStagePreview(
		ToolValidateAWSTerraform,
		"Terraform draft validated",
		params,
		"Next: call deploy_aws_terraform_iteration if you want the preview deploy stub.",
	), false
}

func DeployAWSTerraformTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	_ = ag
	_ = params
	return "Deploy step acknowledged. This workflow is currently a preview stub: it validates the generated Terraform, mirrors progress, and does not make AWS changes.", false
}

func SaveProjectSession(ag *agent.Agent, c *chat.Chat, params map[string]interface{}) (string, bool) {
	_ = ag

	userID := firstNonEmptyString(params, "user_id", "user_email", "email")
	if userID == "" {
		return "user_id (email) is required to save the project session", true
	}

	projectID := firstNonEmptyString(params, "project_id", "session_id", "id")
	if projectID == "" {
		projectID = fmt.Sprintf("project-%d", time.Now().UnixNano())
	}

	projectName := firstNonEmptyString(params, "project_name", "name")
	if projectName == "" {
		projectName = "Saved project session"
	}

	// Persistence is now owned by the central ProjectStore; this tool
	// should not write files directly to avoid conflicting filenames.
	return fmt.Sprintf("acknowledged project session for %s (id=%s)", userID, projectID), false
}

func formatInfraStagePreview(stageName, summary string, params map[string]interface{}, nextStep string) string {
	paramBytes, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		paramBytes = []byte(fmt.Sprintf("{\"marshal_error\": %q}", err.Error()))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s\n", summary))
	sb.WriteString(fmt.Sprintf("Stage: %s\n", stageName))
	sb.WriteString("Mode: preview only; no AWS changes were made.\n")
	sb.WriteString("Inputs:\n")
	sb.WriteString("```json\n")
	sb.Write(paramBytes)
	sb.WriteString("\n```\n")
	if nextStep != "" {
		sb.WriteString(nextStep)
		sb.WriteString("\n")
	}
	return sb.String()
}

func firstNonEmptyString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if str, ok := value.(string); ok {
				trimmed := strings.TrimSpace(str)
				if trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}
