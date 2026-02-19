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
        if !exists && currentToolName != "create_server_tool" {
            return fmt.Errorf("tool %s not found", currentToolName)
        }

        if currentToolName == "create_server_tool" && !ag.ServerGeneration {
            return fmt.Errorf("tool %s not available set it in config to enable server generation", currentToolName)
        }
        
        // Execute the tool
        toolResult, isError := llmprotocol.ExecuteTool(ag, &chat.ToolCall{
            ServerID:   toolInfo.ServerID,
            ToolID:     currentToolName,
            Handler:    toolInfo.Handler,
            Parameters: currentToolParams,
        })
        
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
            "name": "create_server_tool",
            "description": `Generate and deploy a complete Go-based MCP server with one or more custom tools through automated validation.

WHAT IT DOES:
- Generates an entire HTTP server in pure Go
- One server can contain multiple tools
- Automatically compiles, tests, and deploys
- Tools are immediately available for use

VALIDATION PROCESS:
1. Syntax: Go code compiled → compiler errors returned if needed
2. Testing: Each tool tested with YOUR provided test_params → results returned
3. Deploy: On success, server runs and tools are registered

KEY POINTS:
✓ Write pure Go code only
✓ Provide contextual test_params for each tool (for realistic testing)
✓ Each tool is self-contained with its own parameters and test data
✓ Handlers must work with the exact test data you specify

STRUCTURE - tools array with complete tool definitions:
{
  "server_id": "csv_processor",
  "server_description": "Handles CSV parsing and validation",
  "tools": [
    {
      "tool_id": "parse_csv",
      "description": "Parse CSV file content",
      "input_schema": {
        "properties": {
          "content": {"type": "string", "description": "CSV content"},
          "delimiter": {"type": "string", "description": "CSV delimiter"}
        },
        "required": ["content", "delimiter"]
      },
      "handler_code": "var params map[string]interface{}\njson.NewDecoder(r.Body).Decode(&params)\ncontent := params[\"content\"].(string)\n// parse logic...\nresult := map[string]interface{}{\"rows\": parsed}\nw.Header().Set(\"Content-Type\", \"application/json\")\njson.NewEncoder(w).Encode(result)",
      "test_params": {
        "content": "name,age\\nAlice,30\\nBob,25",
        "delimiter": ","
      }
    }
  ]
}

HANDLER CODE TEMPLATE:
  var params map[string]interface{}
  json.NewDecoder(r.Body).Decode(&params)
  
  // Extract parameters
  field1 := params["field_name"].(string)
  field2 := params["field2"].(float64)
  
  // Your business logic here
  result := map[string]interface{}{
    "status": "success",
    "data": processed,
  }
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(result)

TESTING:
- System will call each tool with YOUR test_params
- Your handler must work correctly with those exact values
- If tests fail, you see what went wrong - debug and retry

FEEDBACK:
- Compilation error → Fix Go code → Try again
- Test failed → See test results with your test_params → Debug → Try again
- Success → Server deployed and tools ready!`,
            "input_schema": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "server_id": map[string]interface{}{
                        "type":        "string",
                        "description": "Unique server identifier (snake_case, e.g., 'csv_processor', 'data_analyzer'). One server can contain multiple tools.",
                    },
                    "server_description": map[string]interface{}{
                        "type":        "string",
                        "description": "Description of the server's purpose and what it provides",
                    },
                    "tools": map[string]interface{}{
                        "type":        "array",
                        "description": "Array of tool objects. Each must have: tool_id, description, input_schema, handler_code, test_params",
                        "items": map[string]interface{}{
                            "type": "object",
                            "properties": map[string]interface{}{
                                "tool_id": map[string]interface{}{"type": "string", "description": "Unique tool identifier (snake_case)"},
                                "description": map[string]interface{}{"type": "string", "description": "What this tool does"},
                                "input_schema": map[string]interface{}{"type": "object", "description": "JSON schema with properties and required fields"},
                                "handler_code": map[string]interface{}{"type": "string", "description": "Go handler implementation (complete function body)"},
                                "test_params": map[string]interface{}{"type": "object", "description": "Test data for this specific tool - must match input_schema"},
                            },
                            "required": []string{"tool_id", "description", "input_schema", "handler_code", "test_params"},
                        },
                    },
                },
                "required": []string{"server_id", "server_description", "tools"},
            },
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