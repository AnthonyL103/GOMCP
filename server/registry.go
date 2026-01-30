package registry

import (
	"fmt"
	"strings"
)

//Each agent registry will have its own ID and description

type Registry struct {
	RegistryID string
	Servers map[string]*MCPServer
}

func NewRegistry(
	registryID string,
	servers []*MCPServer
) *Registry {
	registryID = strings.TrimSpace(registryID)

	if registryID == "" {
		panic("RegistryID cannot be empty")
	}

	if len(servers) == 0 {
		panic("At least one server must be provided")
	}

	serverMap := make(map[string]*MCPServer)
	
	for _, server := range servers {
		if server == nil {
			panic("Server cannot be nil")
		}

		serverMap[server.ServerID] = server
	}

	return &Registry{
		RegistryID: registryID,
		Servers:    serverMap,
	}
}

func AddServerToRegistry(
	registry *Registry,
	server *MCPServer,
) {
	if registry == nil {
		panic("Registry cannot be nil")
	}

	if server == nil {
		panic("Server cannot be nil")
	}

	if _, exists := registry.Servers[server.ServerID]; exists {
		panic(fmt.Sprintf("Server with ID '%s' already exists in the registry", server.ServerID))
	}

	registry.Servers[server.ServerID] = server
}

func RemoveServerFromRegistry(
	registry *Registry,
	serverID string,
) {
	if registry == nil {
		panic("Registry cannot be nil")
	}

	if _, exists := registry.Servers[serverID]; !exists {
		panic(fmt.Sprintf("Server with ID '%s' does not exist in the registry", serverID))
	}

	delete(registry.Servers, serverID)
}

func GetServerFromRegistry(
	registry *Registry,
	serverID string,
) *MCPServer {
	if registry == nil {
		panic("Registry cannot be nil")
	}

	if server, exists := registry.Servers[serverID]; exists {
		return server
	}

	panic(fmt.Sprintf("Server with ID '%s' does not exist in the registry", serverID))
}