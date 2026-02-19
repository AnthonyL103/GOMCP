// transport/openai.go
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

type OpenAIProvider struct {
    APIKey      string
    Model       string
    Temperature float32
    MaxTokens   int
}

func NewOpenAIProvider(config *agent.LLMConfig) *OpenAIProvider {
    return &OpenAIProvider{
        APIKey:      config.APIKey,
        Model:       config.Model,
        Temperature: config.Temperature,
        MaxTokens:   config.MaxTokens,
    }
}

func (p *OpenAIProvider) GetProviderName() string {
    return "openai"
}

func (p *OpenAIProvider) SendRequest(c *chat.Chat, ag *agent.Agent, userMessage string) error {
    // Add user message to chat
    c.AddUserMessage(userMessage)
    
    // Extract agent instructions
    agentInstructions := llmprotocol.GetAgentInstructions(ag)
    
    // Extract and format tools
    availableTools := llmprotocol.ExtractTools(ag)
    formattedTools := p.buildTools(availableTools, ag)
    
    // Convert chat history to OpenAI format
    messages := p.buildMessages(c, agentInstructions)
    
    // Create OpenAI API request
    requestBody := map[string]interface{}{
        "model":       p.Model,
        "messages":    messages,
        "temperature": p.Temperature,
        "max_tokens":  p.MaxTokens,
    }
    
    if len(formattedTools) > 0 {
        requestBody["tools"] = formattedTools
    }
    
    // Send request
    response, err := p.sendHTTPRequest(requestBody)
    if err != nil {
        return err
    }
    
    // Parse response - returns tool calls and response text
    responseText, toolCalls, err := p.parseResponse(response)
    if err != nil {
        return err
    }
    
    // Handle tool calls - loop while model wants to use tools
    toolsProcessed := false
    for len(toolCalls) > 0 {
        toolsProcessed = true
        
        // Build ONE assistant message with ALL tool calls
        toolCallsArray := []map[string]interface{}{}
        for toolCallID, toolCall := range toolCalls {
            toolCallMap := toolCall.(map[string]interface{})
            toolCallsArray = append(toolCallsArray, map[string]interface{}{
                "id":   toolCallID,
                "type": "function",
                "function": map[string]interface{}{
                    "name":      toolCallMap["name"].(string),
                    "arguments": p.jsonString(toolCallMap["arguments"].(map[string]interface{})),
                },
            })
        }
        
        // Add single assistant message with all tool calls
        messages = append(messages, map[string]interface{}{
            "role":       "assistant",
            "tool_calls": toolCallsArray,
        })
        
        // Execute and add tool results
        for toolCallID, toolCall := range toolCalls {
            toolCallMap := toolCall.(map[string]interface{})
            currentToolName := toolCallMap["name"].(string)
            currentToolArgs := toolCallMap["arguments"].(map[string]interface{})
            
            // Look up the server that owns this tool
            toolInfo, exists := availableTools[currentToolName]
            if !exists {
                return fmt.Errorf("tool %s not found", currentToolName)
            }
            
            // Execute the tool
            toolResult, isError := llmprotocol.ExecuteTool(ag, &chat.ToolCall{
                ServerID:   toolInfo.ServerID,
                ToolID:     currentToolName,
                Handler:    toolInfo.Handler,
                Parameters: currentToolArgs,
            })
            
            // Add tool result message
            messages = append(messages, map[string]interface{}{
                "role":         "tool",
                "tool_call_id": toolCallID,
                "content":      toolResult,
            })
            
            // Save this tool cycle to chat history
            c.AddAssistantMessage(
                "",
                &chat.ToolCall{
                    ServerID:   toolInfo.ServerID,
                    ToolID:     currentToolName,
                    Handler:    toolInfo.Handler,
                    Parameters: currentToolArgs,
                    Reasoning:  "",
                    ToolUseID:  toolCallID,
                },
                &chat.ToolResult{
                    ServerID:  toolInfo.ServerID,
                    ToolID:    currentToolName,
                    Content:   toolResult,
                    IsError:   isError,
                    ToolUseID: toolCallID,
                },
            )
        }
        
        // Send follow-up request with ALL tool results
        requestBody["messages"] = messages
        response, err = p.sendHTTPRequest(requestBody)
        if err != nil {
            return err
        }
        
        // Update variables for next iteration
        responseText, toolCalls,err = p.parseResponse(response)
        if err != nil {
            return err
        }
        
        // Loop continues if more tool calls exist
    }
    
    // Save final text response (whether tools were used or not)
    if responseText != "" {
        if toolsProcessed {
            // Update last message with final response text
            messages := c.GetMessages()
            if len(messages) > 0 {
                lastMsg := &messages[len(messages)-1]
                lastMsg.Content = responseText
            }
        } else {
            c.AddAssistantMessage(responseText, nil, nil)
        }
    }

    return nil
}
type toolCallInfo struct {
    ID        string
    Name      string
    Arguments map[string]interface{}
}

func (p *OpenAIProvider) buildMessages(c *chat.Chat, systemMessage string) []map[string]interface{} {
    messages := []map[string]interface{}{}
    
    // Add system message first
    if systemMessage != "" {
        messages = append(messages, map[string]interface{}{
            "role":    "system",
            "content": systemMessage,
        })
    }
    
    for _, msg := range c.GetMessages() {
        switch msg.Role {
        case "user":
            messages = append(messages, map[string]interface{}{
                "role":    "user",
                "content": msg.Content,
            })
            
        case "assistant":
            // If message has both ToolCall and ToolResult, expand into tool call format
            if msg.ToolCall != nil && msg.ToolResult != nil {
                // 1. Assistant message with tool_calls
                messages = append(messages, map[string]interface{}{
                    "role": "assistant",
                    "tool_calls": []map[string]interface{}{
                        {
                            "id":   msg.ToolCall.ToolUseID,
                            "type": "function",
                            "function": map[string]interface{}{
                                "name":      msg.ToolCall.ToolID,
                                "arguments": p.jsonString(msg.ToolCall.Parameters),
                            },
                        },
                    },
                })
                
                // 2. Tool message with result
                messages = append(messages, map[string]interface{}{
                    "role":         "tool",
                    "tool_call_id": msg.ToolResult.ToolUseID,
                    "content":      msg.ToolResult.Content,
                })
                
                // 3. Assistant message with final text response if present
                if msg.Content != "" {
                    messages = append(messages, map[string]interface{}{
                        "role":    "assistant",
                        "content": msg.Content,
                    })
                }
            } else if msg.Content != "" {
                // Regular assistant message
                messages = append(messages, map[string]interface{}{
                    "role":    "assistant",
                    "content": msg.Content,
                })
            }
        }
    }
    
    return messages
}

func (p *OpenAIProvider) buildTools(availableTools map[string]llmprotocol.ToolInfo, ag *agent.Agent) []map[string]interface{} {
    tools := []map[string]interface{}{}
    
    for toolID, toolInfo := range availableTools {
        formatschema := toolInfo.Schema
        formatschema["type"] = "object"
        
        tools = append(tools, map[string]interface{}{
            "type": "function",
            "function": map[string]interface{}{
                "name":        toolID,
                "description": toolInfo.Description,
                "parameters":  formatschema,
            },
        })
    }

    if ag.ServerGeneration {
        tools = append(tools, map[string]interface{}{
            "type": "function",
            "function": map[string]interface{}{
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
                "parameters": map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "server_id": map[string]interface{}{
                            "type":        "string",
                            "description": "Unique server identifier (snake_case, e.g., 'csv_parser', 'data_processor'). One server can contain multiple tools.",
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
            },
        })
    }
    
    return tools
}

func (p *OpenAIProvider) sendHTTPRequest(requestBody map[string]interface{}) (map[string]interface{}, error) {
    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+p.APIKey)
    
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

// parseResponse extracts relevant data from OpenAI response
// Returns: (responseText, toolCalls array, error)
func (p *OpenAIProvider) parseResponse(response map[string]interface{}) (string, map[string]interface{}, error) {
    choices, ok := response["choices"].([]interface{})
    if !ok || len(choices) == 0 {
        return "", nil, fmt.Errorf("no choices in response")
    }
    
    choice := choices[0].(map[string]interface{})
    message := choice["message"].(map[string]interface{})
    responseText := ""

    if content, ok := message["content"].(string); ok {
        responseText = content
    }
    
    var toolCalls map[string]interface{}
    if toolCallsRaw, ok := message["tool_calls"].([]interface{}); ok {
        toolCalls = make(map[string]interface{})
        for _, block := range toolCallsRaw {
            blockMap := block.(map[string]interface{})
            function := blockMap["function"].(map[string]interface{})
            
            // Parse arguments JSON string
            var args map[string]interface{}
            if argsStr, ok := function["arguments"].(string); ok {
                json.Unmarshal([]byte(argsStr), &args)
            }
            
            toolCalls[blockMap["id"].(string)] = map[string]interface{}{
                "name":      function["name"].(string),
                "arguments": args,
            }
        }
    }
    
    return responseText, toolCalls, nil
}

// jsonString converts a map to JSON string
func (p *OpenAIProvider) jsonString(data map[string]interface{}) string {
    bytes, _ := json.Marshal(data)
    return string(bytes)
}