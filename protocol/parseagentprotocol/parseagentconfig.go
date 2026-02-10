package parseagentprotocol

import (
    "fmt"
    "os"
    "strings"

    "gopkg.in/yaml.v3"

    "github.com/AnthonyL103/GOMCP/agent"
    "github.com/AnthonyL103/GOMCP/registry"
    "github.com/AnthonyL103/GOMCP/protocol/parseserverprotocol"
)

// AgentConfig represents the root YAML structure
type AgentConfig struct {
    Agents []AgentDefinition `yaml:"agents"`
}

// AgentDefinition represents a single agent
type AgentDefinition struct {
    AgentID     string        `yaml:"agent_id"`
    Description string        `yaml:"description"`
    LLM         LLMConfigYAML `yaml:"llm"`
    Servers     []string      `yaml:"servers"` // Paths to server YAML files
}

// LLMConfigYAML represents LLM settings from YAML
type LLMConfigYAML struct {
    APIKey      string  `yaml:"api_key"`
    Model       string  `yaml:"model"`
    Temperature float32 `yaml:"temperature"`
    MaxTokens   int     `yaml:"max_tokens"`
}

func ParseAgentConfig() (*agent.Agent, error) {
    // Read agent.yaml from project root
    data, err := os.ReadFile("./agentconfig.yaml")
    if err != nil {
        return nil, fmt.Errorf("agent.yaml not found in project root: %w", err)
    }

    // Parse YAML
    var config AgentConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse agent.yaml: %w", err)
    }

    // For now, use the first agent (can extend to support multiple later)
    if len(config.Agents) == 0 {
        return nil, fmt.Errorf("no agents defined in config")
    }

    agentDef := config.Agents[0]

    // Resolve API key (check env variable if needed)
    apiKey := resolveEnvVar(agentDef.LLM.APIKey)
    if apiKey == "" {
        return nil, fmt.Errorf("API key not found for agent %s", agentDef.AgentID)
    }

    // Create LLMConfig for Agent constructor
    LLMConfig := &agent.LLMConfig{
        APIKey:      apiKey,
        Model:       agentDef.LLM.Model,
        Temperature: agentDef.LLM.Temperature,
        MaxTokens:   agentDef.LLM.MaxTokens,
    }

    // Create registry
    reg := registry.NewRegistry()

    // Parse and register each server
    for _, serverPath := range agentDef.Servers {
        server, runtimeconfig, err := parseserverprotocol.ParseServerConfig(serverPath)
        if err != nil {
            return nil, fmt.Errorf("failed to parse server %s: %w", serverPath, err)
        }
        // Store runtime config on the server for later use when executing
        server.RuntimeConfig = runtimeconfig
        err = reg.AddServer(server)
        if err != nil {
            return nil, fmt.Errorf("failed to register server %s: %w", server.ServerID, err)
        }
    }

    // Create agent using your NewAgent constructor
    ag := agent.NewAgent(
        agentDef.AgentID,
        agentDef.Description,
        reg,
        LLMConfig,
    )

    return ag, nil
}

// resolveEnvVar resolves environment variables in format ${VAR_NAME}
func resolveEnvVar(value string) string {
    // Check if value is an env var reference
    if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
        envVar := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
        return os.Getenv(envVar)
    }

    // Return as-is if not an env var
    return value
}

func TestConfigParser() {
    ag, err := ParseAgentConfig()
    if err != nil {
        fmt.Println("ParseAgentConfig error:", err)
        return
    }
    if ag == nil {
        fmt.Println("No agent returned")
        return
    }

    // print agent summary
    details := ag.GetAgentDetails(ag)
    fmt.Printf("Agent ID: %s\nDescription: %s\nServerCount: %d\nToolCount: %d\n",
        details.AgentID, details.Description, details.ServerCount, details.ToolCount)

    // iterate registry servers (map[string]*MCPServer)
    for serverID, server := range ag.Registry.Servers {
        fmt.Printf("Server ID: '%s'\nServer description: '%s'\n", serverID, server.Description)

        for toolName, tool := range server.Tools {
            fmt.Printf("  Tool Key: '%s' Tool Handler '%s' Tool Description '%s' \n", toolName, tool.Handler, tool.Description)
        }
    }
}
