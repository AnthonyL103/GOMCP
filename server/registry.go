package server

import (
	"fmt"
)

type Registry struct {
	Servers map[string]*MCPServer
}

func NewRegistry() *Registry {
	return &Registry{
		Servers: make(map[string]*MCPServer),
	}
}

// AddServer adds a server to the registry
func (r *Registry) AddServer(server *MCPServer) error {
	if server == nil {
		return fmt.Errorf("server cannot be nil")
	}

	if _, exists := r.Servers[server.ServerID]; exists {
		return fmt.Errorf("server with ID '%s' already exists", server.ServerID)
	}

	r.Servers[server.ServerID] = server
	return nil
}

// RemoveServer removes a server from the registry
func (r *Registry) RemoveServer(serverID string) error {
	if _, exists := r.Servers[serverID]; !exists {
		return fmt.Errorf("server with ID '%s' does not exist", serverID)
	}

	delete(r.Servers, serverID)
	return nil
}

// GetServer retrieves a server from the registry
func (r *Registry) GetServer(serverID string) (*MCPServer, error) {
	if server, exists := r.Servers[serverID]; exists {
		return server, nil
	}

	return nil, fmt.Errorf("server with ID '%s' does not exist", serverID)
}

// ListServers returns all servers in the registry
func (r *Registry) ListServers() []*MCPServer {
	servers := make([]*MCPServer, 0, len(r.Servers))
	for _, server := range r.Servers {
		servers = append(servers, server)
	}
	return servers
}

// ExecuteTool executes a tool from a specific server
func (r *Registry) ExecuteTool(serverID, toolID string, input map[string]interface{}) (interface{}, error) {
	server, err := r.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	return server.ExecuteTool(toolID, input)
}