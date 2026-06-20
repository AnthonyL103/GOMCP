package infrageneration

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agent "github.com/AnthonyL103/GOMCP/Agent"
	"github.com/AnthonyL103/GOMCP/chat"
	"github.com/AnthonyL103/GOMCP/store"
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

	project := &store.Project{
		ID:              projectID,
		UserID:          userID,
		Name:            projectName,
		Answers:         coerceMap(params["answers"]),
		Status:          store.StatusComplete,
		Messages:        coerceChatMessages(c),
		TerraformScript: firstNonEmptyString(params, "terraform_script", "terraform", "script"),
		Deployment:      coerceDeployment(params["deployment"]),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if status := firstNonEmptyString(params, "status"); status != "" {
		project.Status = store.ProjectStatus(status)
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

func coerceMap(value interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]interface{}); ok {
		return typed
	}
	return nil
}

func coerceChatMessages(c *chat.Chat) []store.ChatMessage {
	if c == nil {
		return nil
	}

	messages := c.GetMessages()
	out := make([]store.ChatMessage, 0, len(messages))
	for _, msg := range messages {
		out = append(out, store.ChatMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			CreatedAt:  msg.Timestamp,
			ToolCall:   coerceToolCallFromChat(msg.ToolCall),
			ToolResult: coerceToolResultFromChat(msg.ToolResult),
		})
	}
	return out
}

func coerceToolCallFromChat(value *chat.ToolCall) *store.ToolCall {
	if value == nil {
		return nil
	}
	return &store.ToolCall{
		ServerID:   value.ServerID,
		ToolID:     value.ToolID,
		Handler:    value.Handler,
		Parameters: value.Parameters,
		Reasoning:  value.Reasoning,
		ToolUseID:  value.ToolUseID,
	}
}

func coerceToolResultFromChat(value *chat.ToolResult) *store.ToolResult {
	if value == nil {
		return nil
	}
	return &store.ToolResult{
		ServerID:  value.ServerID,
		ToolID:    value.ToolID,
		Content:   value.Content,
		IsError:   value.IsError,
		ToolUseID: value.ToolUseID,
	}
}

func coerceDeployment(value interface{}) *store.DeploymentResult {
	entry, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	return &store.DeploymentResult{
		Status:    firstNonEmptyString(entry, "status"),
		Output:    firstNonEmptyString(entry, "output"),
		Timestamp: coerceTime(entry["timestamp"]),
	}
}

func coerceTime(value interface{}) time.Time {
	if value == nil {
		return time.Time{}
	}
	if typed, ok := value.(time.Time); ok {
		return typed
	}
	if str, ok := value.(string); ok {
		if parsed, err := time.Parse(time.RFC3339, str); err == nil {
			return parsed
		}
		if parsed, err := time.Parse(time.RFC3339Nano, str); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func coerceBool(value interface{}) bool {
	if value == nil {
		return false
	}
	if typed, ok := value.(bool); ok {
		return typed
	}
	if str, ok := value.(string); ok {
		switch strings.ToLower(strings.TrimSpace(str)) {
		case "true", "1", "yes", "y":
			return true
		}
	}
	return false
}

func sanitizeFileComponent(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_' || r == '@':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
