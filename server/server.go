package server

import (
	"fmt"
	"strings"
)

type MCPServer struct {
	ServerID string
	Description string
	Tools map[string]*Tool
}

func NewMCPServer(
	serverID string,
	description string,
	tools []*Tool,
) *MCPServer {
	serverID = strings.TrimSpace(serverID)
	description = strings.TrimSpace(description)
	//validate the inputs
	if serverID == "" || !isString(serverID) {
		panic("ServerID is required and must be a non-empty string")
	}

	if description == "" || !isString(description) {
		panic("Description is required and must be a non-empty string")
	}

	if len(tools) == 0 {
		panic("At least one tool must be provided")
	}

	
	//ensure that no tools are nil as a safeguard and create tool map
	toolMap := make(map[string]*Tool)
	for _, tool := range tools {
		if tool == nil {
			panic("Tool cannot be nil")
		}
		toolMap[tool.Name] = tool
	}

	return &MCPServer{
		ServerID:    serverID,
		Description: description,
		Tools:       toolMap,
	}
}

func (s *MCPServer) AddToolToServer(
	tool *Tool,
) *MCPServer{
	if s == nil {
		panic("Server cannot be nil")
	}
	if tool == nil {
		panic("Tool cannot be nil")
	}
	if _, exists := s.Tools[tool.Name]; exists {
		panic(fmt.Sprintf("Tool with name '%s' already exists in the server", tool.Name))
	}

	s.Tools[tool.Name] = tool
}

func (s *MCPServer) RemoveToolFromServer(
	toolName string,
) *MCPServer {
	if s == nil {
		panic("Server cannot be nil")
	}
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		panic("Tool name cannot be empty")
	}
	if _, exists := s.Tools[toolName]; !exists {
		panic(fmt.Sprintf("Tool with name '%s' does not exist in the server", toolName))
	}
	delete(s.Tools, toolName)
}

func (s *MCPServer) GetToolFromServer(
	toolName string,
) *Tool {
	if s == nil {
		panic("Server cannot be nil")
	}
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		panic("Tool name cannot be empty")
	}

	if tool, exists := s.Tools[toolName]; exists {
		return tool
	}
	panic(fmt.Sprintf("Tool with name '%s' does not exist in the server", toolName))
}