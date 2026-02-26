// transport/anthropic.go
package transport

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    
    "github.com/AnthonyL103/GOMCP/agent"
    "github.com/AnthonyL103/GOMCP/protocol/llmprotocol"
    "github.com/AnthonyL103/GOMCP/chat"
)

type AnthropicProvider struct {
    APIKey      string
    Model       string
    Temperature float32
    MaxTokens   int
}

func NewAnthropicProvider(config *agent.LLMConfig) *AnthropicProvider {
    return &AnthropicProvider{
        APIKey:      config.APIKey,
        Model:       config.Model,
        Temperature: config.Temperature,
        MaxTokens:   config.MaxTokens,
    }
}

func (p *AnthropicProvider) GetProviderName() string {
    return "anthropic"
}

func (p *AnthropicProvider) SendRequest(c *chat.Chat, ag *agent.Agent, userMessage string) error {
    // Add user message to chat
    c.AddUserMessage(userMessage)
    
    // Extract agent instructions
    agentInstructions := llmprotocol.GetAgentInstructions(ag)
    
    // Extract and format tools
    availableTools := llmprotocol.ExtractTools(ag)
    formattedTools := p.buildTools(availableTools, ag)
    
    // Convert chat history to Anthropic format
    messages := p.buildMessages(c)
    
    // Create Anthropic API request
    requestBody := map[string]interface{}{
        "model":       p.Model,
        "max_tokens":  p.MaxTokens,
        "temperature": p.Temperature,
        "system":      agentInstructions,
        "messages":    messages,
    }
    
    if len(formattedTools) > 0 {
        requestBody["tools"] = formattedTools
    }
    
    // Send request
    response, err := p.sendHTTPRequest(requestBody)
    if err != nil {
        return err
    }
    
    // Parse response - returns primitive values
    responseText, toolCallID, toolName, toolParams, stopReason, err := p.parseResponse(response)
    if err != nil {
        return err
    }
    
    // Handle tool calls - loop for multiple sequential tool uses
    toolsProcessed := false
    for stopReason == "tool_use" && toolName != "" {
        toolsProcessed = true
        
        // Save current tool info before it gets overwritten
        currentToolCallID := toolCallID
        currentToolName := toolName
        currentToolParams := toolParams
        
        // Look up the server that owns this tool
        toolInfo, exists := availableTools[currentToolName]
        if !exists {
            if !isServerGenerationToolAnthropic(currentToolName) {
                return fmt.Errorf("tool %s not found", currentToolName)
            }
            toolInfo = llmprotocol.ToolInfo{ServerID: "server_generation", Handler: currentToolName}
        }

        if isServerGenerationToolAnthropic(currentToolName) && !ag.ServerGeneration {
            return fmt.Errorf("tool %s not available; enable server generation in config", currentToolName)
        }


        // Execute the tool
        toolResult, isError := llmprotocol.ExecuteTool(ag, &chat.ToolCall{
            ServerID:   toolInfo.ServerID,
            ToolID:     currentToolName,
            Handler:    toolInfo.Handler,
            Parameters: currentToolParams,
        })

        truncatetoolparams := make(map[string]interface{})
        for k, v := range currentToolParams {
            if strVal, ok := v.(string); ok && len(strVal) > 100 {
                truncatetoolparams[k] = strVal[:100] + "..."
            } else {
                truncatetoolparams[k] = v
            }
        }
        fmt.Println("Calling tool:", currentToolName, "with params:", truncatetoolparams)
        
        // Step 3: Send tool result back to LLM
        messages = p.buildMessages(c)

        // Add assistant's tool_use message
        messages = append(messages, map[string]interface{}{
            "role": "assistant",
            "content": []map[string]interface{}{
                {
                    "type":  "tool_use",
                    "id":    currentToolCallID,
                    "name":  currentToolName,
                    "input": currentToolParams,
                },
            },
        })
    
        // Add user's tool_result message
        messages = append(messages, map[string]interface{}{
            "role": "user",
            "content": []map[string]interface{}{
                {
                    "type":        "tool_result",
                    "tool_use_id": currentToolCallID,
                    "content":     toolResult,
                    "is_error":    isError,
                },
            },
        })
    
        
        // Send follow-up request with tool result
        requestBody["messages"] = messages
        response, err = p.sendHTTPRequest(requestBody)
        if err != nil {
            return err
        }
        
        // Update all variables for next iteration
        responseText, toolCallID, toolName, toolParams, stopReason, err = p.parseResponse(response)
        if err != nil {
            return err
        }
        
        // Save this tool cycle to chat history using CURRENT tool info
        c.AddAssistantMessage(
            responseText,
            &chat.ToolCall{
                ServerID:   toolInfo.ServerID,
                ToolID:     currentToolName,
                Handler:    toolInfo.Handler,
                Parameters: currentToolParams,
                Reasoning:  "",
                ToolUseID:  currentToolCallID,
            },
            &chat.ToolResult{
                ServerID:  toolInfo.ServerID,
                ToolID:    currentToolName,
                Content:   toolResult,
                IsError:   isError,
                ToolUseID: currentToolCallID,
            },
        )
        
        // Loop continues if stopReason is still "tool_use"
    }
    
    // Only save text response if no tools were used
    if !toolsProcessed {
        c.AddAssistantMessage(responseText, nil, nil)
    }

    return nil
}

func (p *AnthropicProvider) buildMessages(c *chat.Chat) []map[string]interface{} {
    messages := []map[string]interface{}{}
    
    for _, msg := range c.GetMessages() {
        switch msg.Role {
        case "user":
            messages = append(messages, map[string]interface{}{
                "role":    "user",
                "content": msg.Content,
            })
            
        case "assistant":
            // If message has both ToolCall and ToolResult, expand into 3 messages
            if msg.ToolCall != nil && msg.ToolResult != nil {
                // 1. Assistant message with tool_use
                messages = append(messages, map[string]interface{}{
                    "role": "assistant",
                    "content": []map[string]interface{}{
                        {
                            "type":  "tool_use",
                            "id":    msg.ToolCall.ToolUseID,
                            "name":  msg.ToolCall.ToolID,
                            "input": msg.ToolCall.Parameters,
                        },
                    },
                })
                
                // 2. User message with tool_result
                messages = append(messages, map[string]interface{}{
                    "role": "user",
                    "content": []map[string]interface{}{
                        {
                            "type":        "tool_result",
                            "tool_use_id": msg.ToolResult.ToolUseID,
                            "content":     msg.ToolResult.Content,
                            "is_error":    msg.ToolResult.IsError,
                        },
                    },
                })
                
                // 3. Assistant message with final text response
                if msg.Content != "" {
                    messages = append(messages, map[string]interface{}{
                        "role": "assistant",
                        "content": []map[string]interface{}{
                            {
                                "type": "text",
                                "text": msg.Content,
                            },
                        },
                    })
                }
            } else {
                // Regular assistant message without complete tool cycle
                content := []map[string]interface{}{}
                
                if msg.Content != "" {
                    content = append(content, map[string]interface{}{
                        "type": "text",
                        "text": msg.Content,
                    })
                }
                
                if len(content) > 0 {
                    messages = append(messages, map[string]interface{}{
                        "role":    "assistant",
                        "content": content,
                    })
                }
            }

        }
    }
    
    return messages
}

func (p *AnthropicProvider) buildTools(availableTools map[string]llmprotocol.ToolInfo, ag *agent.Agent) []map[string]interface{} {
    tools := []map[string]interface{}{}
    
    for toolID, toolInfo := range availableTools {
        formatschema := toolInfo.Schema
        formatschema["type"] = "object"
        tools = append(tools, map[string]interface{}{
            "name":         toolID,
            "description":  toolInfo.Description,
            "input_schema": formatschema,
        })
    }

    if ag.ServerGeneration {
        tools = append(tools, map[string]interface{}{
            "name": "generate_server_code",
            "description": "Generate and validate Go server code for a custom tool server. Auto-includes: net/http, encoding/json, log. Returns a process_id for subsequent deployment steps.",
            "input_schema": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "server_id": map[string]interface{}{"type": "string", "description": "Unique server identifier (snake_case)"},
                    "server_description": map[string]interface{}{"type": "string", "description": "Description of the server's purpose and capabilities"},
                    "tools": map[string]interface{}{
                        "type": "array",
                        "description": "Array of tool objects to implement. Each must have: tool_id, description, input_schema (JSON schema), and handler_code (Go function body)",
                        "items": map[string]interface{}{
                            "type": "object",
                            "properties": map[string]interface{}{
                                "tool_id": map[string]interface{}{"type": "string", "description": "Unique tool identifier (snake_case). Will be POST /execute/{tool_id}"},
                                "description": map[string]interface{}{"type": "string", "description": "Clear description of what this tool does"},
                                "input_schema": map[string]interface{}{"type": "object", "description": "JSON Schema with 'properties' and 'required' arrays. Input will be decoded from JSON request body."},
                                "handler_code": map[string]interface{}{"type": "string", "description": "Go handler code (function body only). Access request body via 'var params map[string]interface{}' with json.Unmarshal. Write JSON response with w.Write."},
                            },
                            "required": []string{"tool_id", "description", "input_schema", "handler_code"},
                        },
                    },
                    "imports": map[string]interface{}{
                        "type": "array",
                        "description": "Optional additional Go imports needed for handler code (e.g. 'crypto/md5', 'strconv', 'strings'). Do not include: net/http, encoding/json, log (auto-included).",
                        "items": map[string]interface{}{"type": "string"},
                    },
                },
                "required": []string{"server_id", "server_description", "tools"},
            },
        })

        tools = append(tools, map[string]interface{}{
            "name": "deploy_and_test_tools",
            "description": "Compile, start the server binary, and verify all tools work with test_params. Must have a test_params object for every tool defined in generate_server_code. On test failure, artifacts are automatically cleaned up. On success, proceed to deploy_and_register_server.",
            "input_schema": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "process_id": map[string]interface{}{"type": "string", "description": "Process ID returned from generate_server_code"},
                    "tool_tests": map[string]interface{}{
                        "type": "array",
                        "description": "Test cases for each tool. Must match the tools defined in generate_server_code. Each test will POST to /execute/{tool_id} with test_params as JSON body.",
                        "items": map[string]interface{}{
                            "type": "object",
                            "properties": map[string]interface{}{
                                "tool_id": map[string]interface{}{"type": "string", "description": "Tool ID to test"},
                                "test_params": map[string]interface{}{"type": "object", "description": "Test params for the tool"},
                            },
                            "required": []string{"tool_id", "test_params"},
                        },
                    },
                },
                "required": []string{"process_id", "tool_tests"},
            },
        })

        tools = append(tools, map[string]interface{}{
            "name": "deploy_and_register_server",
            "description": "Register the tested server into the agent's registry and start the final server process. The server will remain running and available for tool calls. Clean up source files after this step with cleanup_server_generation.",
            "input_schema": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "process_id": map[string]interface{}{"type": "string", "description": "Process ID from deploy_and_test_tools (after successful tests)"},
                },
                "required": []string{"process_id"},
            },
        })

        tools = append(tools, map[string]interface{}{
            "name": "cleanup_server_generation",
            "description": "Clean up temporary files after server deployment. Removes the Go source file (binary is kept and running). Call this after deploy_and_register_server completes successfully.",
            "input_schema": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "process_id": map[string]interface{}{"type": "string", "description": "Process ID to clean up (returned from generate_server_code)"},
                },
                "required": []string{"process_id"},
            },
        })
    }

    tools = append(tools, map[string]interface{}{
        "name": "delete_server_tool",
        "description": "Permanently delete a previously generated and deployed server. Stops the running process, removes files, and unregisters from the agent's registry.",
        "input_schema": map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "server_id": map[string]interface{}{
                    "type":        "string",
                    "description": "The unique identifier of the server to delete (e.g., 'csv_processor')",
                },
            },
            "required": []string{"server_id"},
        },
    })
    

    return tools
}

func (p *AnthropicProvider) sendHTTPRequest(requestBody map[string]interface{}) (map[string]interface{}, error) {
    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", p.APIKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
    }
    
    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    
    return response, nil
}

// parseResponse extracts relevant data from Anthropic's response
// Returns: (responseText, toolCallID, toolName, toolParams, stopReason, error)
func (p *AnthropicProvider) parseResponse(response map[string]interface{}) (string, string, string, map[string]interface{}, string, error) {
    content, ok := response["content"].([]interface{})
    if !ok || len(content) == 0 {
        return "", "", "", nil, "", fmt.Errorf("no content in response")
    }
    
    stopReason := response["stop_reason"].(string)
    responseText := ""
    toolCallID := ""
    toolName := ""
    var toolParams map[string]interface{}
    
    for _, block := range content {
        blockMap := block.(map[string]interface{})
        blockType := blockMap["type"].(string)
        
        switch blockType {
        case "text":
            responseText = blockMap["text"].(string)

        case "tool_use":
            toolCallID = blockMap["id"].(string)
            toolName = blockMap["name"].(string)
            toolParams = blockMap["input"].(map[string]interface{})
        }
    }

    return responseText, toolCallID, toolName, toolParams, stopReason, nil
}

// isServerGenerationTool returns true for tools handled in-process.
func isServerGenerationToolAnthropic(name string) bool {
    switch name {
    case "generate_server_code", "deploy_and_test_tools", "deploy_and_register_server", "cleanup_server_generation", "delete_server_tool":
        return true
    default:
        return false
    }
}