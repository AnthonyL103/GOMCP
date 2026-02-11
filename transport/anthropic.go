// transport/anthropic.go
package transport

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "github.com/AnthonyL103/GOMCP/agent"
    "github.com/AnthonyL103/GOMCP/protocol/llmprotocol"
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

func (p *AnthropicProvider) SendRequest(chat *Chat, ag *agent.Agent, userMessage string) (*llmprotocol.LLMResponse, error) {
    // Add user message to chat
    chat.AddUserMessage(userMessage)
    
    // Extract agent instructions
    agentInstructions := llmprotocol.GetAgentInstructions(ag)
    
    // Extract and format tools
    availableTools := llmprotocol.ExtractTools(ag)
    formattedTools := p.buildTools(availableTools)
    
    // Convert chat history to Anthropic format
    messages := p.buildMessages(chat)
    
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
        return nil, err
    }
    
    // Parse response
    llmResponse, err := p.parseResponse(response)
    if err != nil {
        return nil, err
    }
    
    // Handle tool call - multi-step process
    if llmResponse.ToolCall != nil {
        // Look up the server that owns this tool
        toolInfo, exists := availableTools[llmResponse.ToolCall.ToolName]
        if !exists {
            return nil, fmt.Errorf("tool %s not found", llmResponse.ToolCall.ToolName)
        }
        
        // Step 1: Add the tool call message
        chat.AddAssistantMessage("", &ToolCall{
            ServerID:   toolInfo.ServerID,  // Get from toolInfo!
            ToolID:     llmResponse.ToolCall.ToolName,
            Parameters: llmResponse.ToolCall.Parameters,
            Reasoning:  llmResponse.ToolCall.Reasoning,
        }, nil)
        
        // Step 2: Execute the tool with serverID
        toolResult, isError := llmprotocol.ExecuteTool(ag, toolInfo.ServerID, llmResponse.ToolCall)
        // Step 3: Send tool result back to LLM to get final response
        // Add tool result to messages for next call
        messages = p.buildMessages(chat)
        messages = append(messages, map[string]interface{}{
            "role": "user",
            "content": []map[string]interface{}{
                {
                    "type":        "tool_result",
                    "tool_use_id": llmResponse.ToolCall.ToolCallID, // Need to track this
                    "content":     toolResult,
                    "is_error":    isError,
                },
            },
        })
        
        // Send follow-up request with tool result
        requestBody["messages"] = messages
        response2, err := p.sendHTTPRequest(requestBody)
        if err != nil {
            return nil, err
        }
        
        finalResponse, err := p.parseResponse(response2)
        if err != nil {
            return nil, err
        }
        
        // Step 4: Add final assistant message with tool result
        chat.AddAssistantMessage(
        finalResponse.ResponseText,
        nil,
        &ToolResult{
            ServerID: toolInfo.ServerID,  // Use the same serverID
            ToolID:   llmResponse.ToolCall.ToolName,
            Content:  toolResult,
            IsError:  isError,
        },
        )
        
        return finalResponse, nil
    }
    
    // No tool call - just text response
    chat.AddAssistantMessage(llmResponse.ResponseText, nil, nil)
    return llmResponse, nil
}

func (p *AnthropicProvider) buildMessages(chat *Chat) []map[string]interface{} {
    messages := []map[string]interface{}{}
    
    for _, msg := range chat.GetMessages() {
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
                    "type":  "tool_use",
                    "id":    msg.ToolCall.ToolID + "_" + fmt.Sprint(msg.Timestamp.Unix()),
                    "name":  msg.ToolCall.ToolID,
                    "input": msg.ToolCall.Parameters,
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
                            "tool_use_id": msg.ToolResult.ToolID + "_" + fmt.Sprint(msg.Timestamp.Unix()),
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
        tools = append(tools, map[string]interface{}{
            "name":         toolID,
            "description":  toolInfo.Description,
            "input_schema": toolInfo.Schema, // Already a map, no unmarshal needed!
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

func (p *AnthropicProvider) parseResponse(response map[string]interface{}) (*llmprotocol.LLMResponse, error) {
    content, ok := response["content"].([]interface{})
    if !ok || len(content) == 0 {
        return nil, fmt.Errorf("no content in response")
    }
    
    llmResponse := &llmprotocol.LLMResponse{
        StopReason: response["stop_reason"].(string),
    }
    
    for _, block := range content {
        blockMap := block.(map[string]interface{})
        blockType := blockMap["type"].(string)
        
        switch blockType {
        case "text":
            llmResponse.ResponseText = blockMap["text"].(string)
            
        case "tool_use":
            llmResponse.ToolCall = &llmprotocol.ToolCall{
                ToolCallID: blockMap["id"].(string), // Save the ID!
                ToolName:   blockMap["name"].(string),
                Parameters: blockMap["input"].(map[string]interface{}),
                Reasoning:  "", // Anthropic doesn't return reasoning separately
            }
            llmResponse.StopReason = "tool_use"
        }
    }
    
    return llmResponse, nil
}