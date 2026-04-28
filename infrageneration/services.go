package infrageneration

import (
	"encoding/json"
	"fmt"
	"strings"

	agent "github.com/AnthonyL103/GOMCP/Agent"
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
	return "true", false
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
