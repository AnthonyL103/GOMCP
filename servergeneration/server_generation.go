package servergeneration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/AnthonyL103/GOMCP/agent"
	"github.com/AnthonyL103/GOMCP/registry"
	"github.com/AnthonyL103/GOMCP/server"
	"github.com/AnthonyL103/GOMCP/tool"
)

// GeneratedServer tracks a dynamically created server in go only
type GeneratedServer struct {
	ServerID    string
	Port        int
	FilePath    string
	BinaryPath  string
	Process     *exec.Cmd
	Tools       map[string]*tool.Tool
	Running     bool
	CreatedAt   time.Time
}

// GeneratedTool represents a single tool definition from the LLM
type GeneratedTool struct {
	ToolID      string                 `json:"tool_id"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	HandlerCode string                 `json:"handler_code"`
	TestParams  map[string]interface{} `json:"test_params"`
}

// ServerManager manages all generated servers
type ServerManager struct {
	servers                  map[string]*GeneratedServer
	mu                       sync.RWMutex
	nextPort                 int
	serverGeneratedThisCall  bool // Track if a server was already generated in this LLM call
}

var manager = &ServerManager{
	servers:                 make(map[string]*GeneratedServer),
	nextPort:                9000, // Start dynamic servers at port 9000
	serverGeneratedThisCall: false,
}

// GenerateServerTool is the main entry point called by the LLM
// Returns (result_message, is_error)
func GenerateServerTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	// Check if a server was already generated in this request
	manager.mu.Lock()
	if manager.serverGeneratedThisCall {
		manager.mu.Unlock()
		return "Error: A server has already been generated and deployed in this request. Only ONE server per request is allowed. The first server was deployed and this request is rejected.", true
	}
	manager.mu.Unlock()

	// Safety check: limit total servers to prevent runaway generation
	manager.mu.RLock()
	serverCount := len(manager.servers)
	manager.mu.RUnlock()
	
	if serverCount >= 5 {
		return "Error: Maximum 5 servers allowed. Please stop some servers before creating new ones.", true
	}

	// Parse parameters
	serverID, _ := params["server_id"].(string)
	serverDescription, _ := params["server_description"].(string)
	
	// Parse tools array
	toolsRaw := params["tools"].([]interface{})
	if len(toolsRaw) == 0 {
		return "Error: at least one tool is required in the tools array", true
	}

	// Convert tools to GeneratedTool structs
	generatedTools := make([]GeneratedTool, 0)
	for _, t := range toolsRaw {
		toolMap := t.(map[string]interface{})
		generatedTools = append(generatedTools, GeneratedTool{
			ToolID:      toolMap["tool_id"].(string),
			Description: toolMap["description"].(string),
			InputSchema: toolMap["input_schema"].(map[string]interface{}),
			HandlerCode: toolMap["handler_code"].(string),
			TestParams:  toolMap["test_params"].(map[string]interface{}),
		})
	}

	if serverID == "" || len(generatedTools) == 0 {
		return "Error: server_id and tools are required", true
	}

	// Step 1: Generate the server code
	port := manager.allocatePort()
	filePath, err := generateServerCode(serverID, generatedTools, port)
	if err != nil {
		return fmt.Sprintf("Failed to generate server code: %v", err), true
	}

	fmt.Printf("Generated server code at %s\n", filePath)

	// Step 2: Validate syntax
	binaryPath := resolveBinaryPath(strings.TrimSuffix(filePath, ".go"))
	syntaxResult, syntaxErr := ValidateSyntax(filePath, binaryPath)
	if syntaxErr != nil {
		return fmt.Sprintf("SYNTAX VALIDATION FAILED:\n%s\n\nPlease fix the Go code and try again.", syntaxResult), true
	}

	fmt.Printf("Syntax validation passed, compiled binary at %s\n", binaryPath)	

	// Step 3: Start the server for testing, cmd is the test server process that we will kill after testing
	cmd, err := startServerProcess(binaryPath, port)
	if err != nil {
		return fmt.Sprintf("Failed to start server for testing: %v", err), true
	}

	fmt.Printf("Started server for testing on port %d\n", port)

	// Step 4: Test all tools with their specific test_params
	testResult, testErr := TestTools(port, generatedTools)
	
	// Stop test server
	if cmd.Process != nil {
		cmd.Process.Kill()
	}

	if testErr != nil {
		// Clean up binary and source file
		os.Remove(binaryPath + ".exe")
		os.Remove(filePath)
		return fmt.Sprintf("TOOL TEST FAILED:\n%s\n\nPlease fix the handler logic and try again.", testResult), true
	}

	// Step 5: Register the server and tools
	toolObjs := make([]*tool.Tool, 0)
	for _, genTool := range generatedTools {
		toolObj := createToolObject(genTool.ToolID, genTool.Description, genTool.InputSchema)
		toolObjs = append(toolObjs, toolObj)
	}

	fmt.Printf("All tools passed testing:\n%s\nRegistering server in the agent registry...\n", testResult)

	// Reuse the same port/binary for permanent deployment
	err = AddToRegistry(ag.Registry, serverID, serverDescription, port, toolObjs)
	if err != nil {
		return fmt.Sprintf("Failed to register server: %v", err), true
	}

	fmt.Printf("Server '%s' registered successfully with %d tools\n", serverID, len(toolObjs))

	fmt.Printf("current registry state: %+v\n", ag.Registry.Servers)

	fmt.Printf("Generated server binary perm: %s, Port: %d\n", binaryPath, port)
	// Step 6: Start the server permanently
	cmd, err = startServerProcess(binaryPath, port)
	if err != nil {
		return fmt.Sprintf("Failed to start server: %v", err), true
	}

	// Track the generated server
	manager.mu.Lock()
	toolMap := make(map[string]*tool.Tool)
	for _, toolObj := range toolObjs {
		toolMap[toolObj.ToolID] = toolObj
	}
	manager.servers[serverID] = &GeneratedServer{
		ServerID:   serverID,
		Port:       port,
		FilePath:   filePath,
		BinaryPath: binaryPath,
		Process:    cmd,
		Tools:      toolMap,
		Running:    true,
		CreatedAt:  time.Now(),
	}
	manager.mu.Unlock()

	toolNames := make([]string, len(generatedTools))
	for i, t := range generatedTools {
		toolNames[i] = t.ToolID
	}

	// Mark that a server has been generated in this request
	manager.mu.Lock()
	manager.serverGeneratedThisCall = true
	manager.mu.Unlock()

	return fmt.Sprintf("SUCCESS! Server '%s' created and deployed.\n\n"+
		"- Tools: %s\n"+
		"- Port: %d\n"+
		"- Status: Running\n"+
		"- File: %s\n\n"+
		"All tools are now available for immediate use!", 
		serverID, strings.Join(toolNames, ", "), port, filePath), false
}

// ValidateSyntax validates Go code by attempting to compile it
// Returns (error_output, error)
func ValidateSyntax(sourcePath, binaryPath string) (string, error) {
	cmd := exec.Command("go", "build", "-o", binaryPath, sourcePath)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return string(output), fmt.Errorf("compilation failed")
	}
	
	return "Syntax validation passed ✓", nil
}

// TestTools tests all tools with their provided test_params
// Returns (result_description, error)
func TestTools(port int, tools []GeneratedTool) (string, error) {
	// Wait for server to be ready with retries
	healthURL := fmt.Sprintf("http://localhost:%d/execute/", port)
	maxRetries := 10
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			break // Server is ready
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	
	if lastErr != nil && maxRetries > 0 {
		return fmt.Sprintf("Server failed to start after %d retries: %v", maxRetries, lastErr), fmt.Errorf("server startup timeout")
	}

	var testResults strings.Builder
	testResults.WriteString("TESTING TOOLS:\n")

	for _, tool := range tools {
		url := fmt.Sprintf("http://localhost:%d/execute/%s", port, tool.ToolID)

		// Use the LLM-provided test params
		testPayload := tool.TestParams
		jsonData, err := json.Marshal(testPayload)
		if err != nil {
			return fmt.Sprintf("Failed to marshal test payload for %s: %v", tool.ToolID, err), fmt.Errorf("marshal failed")
		}

		// Make test request with retries
		var resp *http.Response
		var lastConnErr error
		for attempt := 0; attempt < 3; attempt++ {
			resp, lastConnErr = http.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if lastConnErr == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		
		if lastConnErr != nil {
			return fmt.Sprintf("Tool '%s': Failed to connect to endpoint: %v", tool.ToolID, lastConnErr), fmt.Errorf("connection failed")
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Sprintf("Tool '%s': Failed to read response: %v", tool.ToolID, err), fmt.Errorf("read failed")
		}

		// Check if response is valid JSON
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Sprintf("Tool '%s': Returned invalid JSON: %s", tool.ToolID, string(body)), fmt.Errorf("invalid JSON")
		}

		// Check for HTTP errors
		if resp.StatusCode >= 400 {
			return fmt.Sprintf("Tool '%s': HTTP error %d: %s", tool.ToolID, resp.StatusCode, string(body)), fmt.Errorf("HTTP error")
		}

		testResults.WriteString(fmt.Sprintf("\n✓ %s\n", tool.ToolID))
		testResults.WriteString(fmt.Sprintf("  Response: %s\n", string(body)))
	}

	return testResults.String(), nil
}

// AddToRegistry registers a generated server in the agent's registry
func AddToRegistry(reg *registry.Registry, serverID, description string, port int, toolObjs []*tool.Tool) error {
	runtimeConfig := &server.RuntimeConfig{
		Type:    "http",
		Command: "",
		Args:    []string{},
		Port:    port,
	}

	mcpServer := server.NewMCPServer(
		serverID,
		description,
		toolObjs,
		runtimeConfig,
	)

	return reg.AddServer(mcpServer)
}

// DeleteFromRegistry removes a generated server from the registry
func DeleteServerTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	//unlock registry to delete server, defer re-lock until end of function
	manager.mu.Lock()
	defer manager.mu.Unlock()

	serverID, ok := params["server_id"].(string)
	if !ok {
		return "delete_server_tool requires 'server_id' parameter", true
	}

	genServer, exists := manager.servers[serverID]
	if !exists {
		return fmt.Sprintf("server '%s' not found", serverID), true
	}

	os.Remove(genServer.FilePath)
	os.Remove(genServer.BinaryPath + ".exe")
	if genServer.Process != nil && genServer.Process.Process != nil {
		genServer.Process.Process.Kill()
	}
	// Remove from manager
	delete(manager.servers, serverID)

	return fmt.Sprintf("Server '%s' deleted successfully", serverID), false
}

// startServerProcess starts a compiled server binary
func startServerProcess(binaryPath string, port int) (*exec.Cmd, error) {
	actualPath := resolveBinaryPath(binaryPath)
	if _, err := os.Stat(actualPath); err != nil {
		return nil, fmt.Errorf("file not found at %s: %v", actualPath, err)
	}

	cmd := exec.Command(actualPath)
	
	// Don't redirect stdout/stderr to parent - let server run independently
	// but still capture for debugging if needed
	cmd.Stdout = nil
	cmd.Stderr = nil
	
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %v", err)
	}
	
	log.Printf("Started generated server on port %d (PID: %d)", port, cmd.Process.Pid)
	return cmd, nil
}

// resolveBinaryPath ensures the correct executable path for the current OS.
func resolveBinaryPath(basePath string) string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(basePath), ".exe") {
		return basePath + ".exe"
	}
	return basePath
}

// generateServerCode creates the Go source file for a new server
// generateServerCode creates the Go source file for a new server with multiple tools
func generateServerCode(serverID string, tools []GeneratedTool, port int) (string, error) {
	// Create generated_servers directory
	genDir := "generated_servers"
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"log\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("func main() {\n")
	
	// Register handler for each tool
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("\thttp.HandleFunc(\"/execute/%s\", handle%s)\n", tool.ToolID, toPascalCase(tool.ToolID)))
	}
	
	sb.WriteString(fmt.Sprintf("\n\tlog.Printf(\"Starting %s on port %d\")\n", serverID, port))
	sb.WriteString(fmt.Sprintf("\tlog.Fatal(http.ListenAndServe(\":%d\", nil))\n", port))
	sb.WriteString("}\n\n")

	// Generate handler function for each tool
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("func handle%s(w http.ResponseWriter, r *http.Request) {\n", toPascalCase(tool.ToolID)))
		sb.WriteString("\tif r.Method != \"POST\" {\n")
		sb.WriteString("\t\thttp.Error(w, \"Method not allowed\", http.StatusMethodNotAllowed)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		
		// Insert the LLM-provided handler code
		sb.WriteString(indentCode(tool.HandlerCode, 1))
		sb.WriteString("\n}\n\n")
	}

	// Write to file
	filePath := filepath.Join(genDir, fmt.Sprintf("%s_server.go", serverID))
	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

// createToolObject creates a Tool object for registration
func createToolObject(toolID, description string, inputSchema map[string]interface{}) *tool.Tool {
	properties, _ := inputSchema["properties"].(map[string]interface{})
	required, _ := inputSchema["required"].([]interface{})

	// Convert to tool.PropertySchema format
	propSchemas := make(map[string]tool.PropertySchema)
	for key, val := range properties {
		propMap, _ := val.(map[string]interface{})
		propType, _ := propMap["type"].(string)
		propDesc, _ := propMap["description"].(string)
		
		propSchemas[key] = tool.PropertySchema{
			Type:        propType,
			Description: propDesc,
		}
	}

	// Convert required array
	requiredFields := make([]string, 0, len(required))
	for _, req := range required {
		if reqStr, ok := req.(string); ok {
			requiredFields = append(requiredFields, reqStr)
		}
	}

	return tool.NewTool(
		toolID,
		description,
		tool.JSONSchema{
			Properties: propSchemas,
			Required:   requiredFields,
		},
		fmt.Sprintf("/execute/%s", toolID),
	)
}

// allocatePort returns the next available port
func (sm *ServerManager) allocatePort() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	port := sm.nextPort
	sm.nextPort++
	return port
}

// deallocatePort marks a port as available for reuse
func (sm *ServerManager) deallocatePort(port int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Simply decrement nextPort if this was the last allocated port
	if port == sm.nextPort-1 {
		sm.nextPort--
	}
}

// toPascalCase converts snake_case to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// indentCode adds indentation to code
func indentCode(code string, level int) string {
	indent := strings.Repeat("\t", level)
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}







