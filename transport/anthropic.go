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
    formattedTools := p.buildTools(availableTools)
    
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
    
    // Handle tool call - multi-step process
    if stopReason == "tool_use" && toolName != "" {
        // Look up the server that owns this tool
        toolInfo, exists := availableTools[toolName]
        if !exists {
            return fmt.Errorf("tool %s not found", toolName)
        }
        
        // Step 1: Add the tool call message
        c.AddAssistantMessage("", &chat.ToolCall{
            ServerID:   toolInfo.ServerID,
            ToolID:     toolName,
            Handler:    toolInfo.Handler,
            Parameters: toolParams,
            Reasoning:  "",
            ToolUseID:  toolCallID,
        }, nil)
        
        // Step 2: Execute the tool
        toolResult, isError := llmprotocol.ExecuteTool(ag, &chat.ToolCall{
            ServerID:   toolInfo.ServerID,
            ToolID:     toolName,
            Handler:    toolInfo.Handler,
            Parameters: toolParams,
        })
        
        // Step 3: Send tool result back to LLM
        messages = p.buildMessages(c)
        messages = append(messages, map[string]interface{}{
            "role": "user",
            "content": []map[string]interface{}{
                {
                    "type":        "tool_result",
                    "tool_use_id":    toolCallID,
                    "content":     toolResult,
                    "is_error":    isError,
                },
            },
        })
        
        // Send follow-up request with tool result
        requestBody["messages"] = messages
        response2, err := p.sendHTTPRequest(requestBody)
        if err != nil {
            return err
        }
        
        finalResponseText, _, _, _, _, err := p.parseResponse(response2)
        if err != nil {
            return err
        }
        
        // Step 4: Add final assistant message with tool result
        c.AddAssistantMessage(
            finalResponseText,
            nil,
            &chat.ToolResult{
                ServerID: toolInfo.ServerID,
                ToolID:   toolName,
                Content:  toolResult,
                IsError:  isError,
                ToolUseID: toolCallID,
            },
        )
        
        return nil
    }
    
    // No tool call - just text response
    c.AddAssistantMessage(responseText, nil, nil)
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
            content := []map[string]interface{}{}
            
            // Add text if present
            if msg.Content != "" {
                content = append(content, map[string]interface{}{
                    "type": "text",
                    "text": msg.Content,
                })
            }
            
            // Add tool use if this message has a tool call
            if msg.ToolCall != nil {
                content = append(content, map[string]interface{}{
                    "type":   "tool_use",
                    "id": msg.ToolCall.ToolUseID,
                    "name":   msg.ToolCall.ToolID,
                    "input":  msg.ToolCall.Parameters,
                })
            }
            
            // Only add if there's content
            if len(content) > 0 {
                messages = append(messages, map[string]interface{}{
                    "role":    "assistant",
                    "content": content,
                })
            }
            
            // If this message has a tool result, add it as user message
            if msg.ToolResult != nil {
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
            }

        }
    }
    
    return messages
}

func (p *AnthropicProvider) buildTools(availableTools map[string]llmprotocol.ToolInfo) []map[string]interface{} {
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