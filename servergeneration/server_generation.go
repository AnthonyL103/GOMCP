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

const (
	ToolGenerateServerCode      = "generate_server_code"
	ToolDeployAndTestTools      = "deploy_and_test_tools"
	ToolDeployAndRegister       = "deploy_and_register_server"
	ToolCleanupServerGeneration = "cleanup_server_generation"
	ToolDeleteServer            = "delete_server_tool"
)

type GenerationStage string

const (
	StageInit          GenerationStage = "init"
	StageCodeGenerated GenerationStage = "code_generated"
	StageTested        GenerationStage = "tested"
	StageDeployed      GenerationStage = "deployed"
	StageCleaned       GenerationStage = "cleaned"
	StageFailed        GenerationStage = "failed"
)

// GeneratedServer tracks a dynamically created server in go only.
type GeneratedServer struct {
	ServerID   string
	Port       int
	FilePath   string
	BinaryPath string
	Process    *exec.Cmd
	Tools      map[string]*tool.Tool
	Running    bool
	CreatedAt  time.Time
}

// GeneratedTool represents a single tool definition from the LLM.
type GeneratedTool struct {
	ToolID      string                 `json:"tool_id"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	HandlerCode string                 `json:"handler_code"`
	TestParams  map[string]interface{} `json:"test_params"`
}

// ServerGenerationProcess tracks an in-progress server generation flow.
type ServerGenerationProcess struct {
	ID          string
	ServerID    string
	Description string
	Tools       []GeneratedTool
	Port        int
	FilePath    string
	BinaryPath  string
	Stage       GenerationStage
	CreatedAt   time.Time
	LastError   string
}

// ServerManager manages generated servers and in-flight processes.
type ServerManager struct {
	servers       map[string]*GeneratedServer
	processes     map[string]*ServerGenerationProcess
	mu            sync.RWMutex
	nextPort      int
	nextProcessID int64
}

var manager = &ServerManager{
	servers:   make(map[string]*GeneratedServer),
	processes: make(map[string]*ServerGenerationProcess),
	nextPort:  9000,
}

// GenerateServerCodeTool creates server code and validates syntax.
func GenerateServerCodeTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	serverID, serverDescription, tools, imports, err := parseGenerateParams(params)
	if err != nil {
		return err.Error(), true
	}

	manager.mu.RLock()
	serverCount := len(manager.servers)
	manager.mu.RUnlock()
	if serverCount >= 5 {
		return "Error: maximum 5 servers allowed. Please stop some servers before creating new ones.", true
	}

	port := manager.allocatePort()
	processID := manager.newProcessID()

	genDir := "generated_servers"
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return fmt.Sprintf("Failed to create generated_servers directory: %v", err), true
	}

	filePath := filepath.Join(genDir, fmt.Sprintf("%s_server.go", serverID))
	source := buildServerSource(serverID, port, tools, imports)
	if err := os.WriteFile(filePath, []byte(source), 0644); err != nil {
		return fmt.Sprintf("Failed to write server code: %v", err), true
	}

	binaryPath := resolveBinaryPath(strings.TrimSuffix(filePath, ".go"))
	syntax, syntaxErr := ValidateSyntax(filePath, binaryPath)
	if syntaxErr != nil {
		cleanupArtifacts(filePath, binaryPath)
		return fmt.Sprintf("SYNTAX VALIDATION FAILED:\n%s\n\nPlease fix the Go code and try again.", syntax), true
	}

	process := &ServerGenerationProcess{
		ID:          processID,
		ServerID:    serverID,
		Description: serverDescription,
		Tools:       tools,
		Port:        port,
		FilePath:    filePath,
		BinaryPath:  binaryPath,
		Stage:       StageCodeGenerated,
		CreatedAt:   time.Now(),
	}

	manager.mu.Lock()
	manager.processes[processID] = process
	manager.mu.Unlock()

	return fmt.Sprintf(
		"SUCCESS: Server code generated and validated.\n\n"+
			"- Process ID: %s\n"+
			"- Server ID: %s\n"+
			"- Port: %d\n"+
			"- File: %s\n\n"+
			"Next: call deploy_and_test_tools with the process_id and tool tests.",
		processID, serverID, port, filePath,
	), false
}

// DeployAndTestToolsTool starts the server, runs tests, and tears down the test process.
func DeployAndTestToolsTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	process, err := getProcessFromParams(params)
	if err != nil {
		return err.Error(), true
	}
	if process.Stage != StageCodeGenerated {
		return fmt.Sprintf("Invalid state: expected %s but got %s", StageCodeGenerated, process.Stage), true
	}

	toolTests, err := parseToolTests(params)
	if err != nil {
		return err.Error(), true
	}

	missing := assignTestParams(process.Tools, toolTests)
	if len(missing) > 0 {
		return fmt.Sprintf("Missing test_params for tools: %s", strings.Join(missing, ", ")), true
	}

	cmd, err := startServerProcess(process.BinaryPath)
	if err != nil {
		cleanupProcess(process)
		return fmt.Sprintf("Failed to start test server: %v", err), true
	}
	defer stopProcess(cmd)

	if err := waitForServer(process.Port, 10, 200*time.Millisecond); err != nil {
		cleanupProcess(process)
		return fmt.Sprintf("Server failed to start: %v", err), true
	}

	results, err := runToolTests(process.Port, process.Tools)
	if err != nil {
		cleanupProcess(process)
		return fmt.Sprintf("TOOL TEST FAILED:\n%s", results), true
	}

	manager.mu.Lock()
	process.Stage = StageTested
	manager.mu.Unlock()

	return results + "\n\nNext: call deploy_and_register_server with the process_id.", false
}

// DeployAndRegisterServerTool registers and starts the final server.
func DeployAndRegisterServerTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	process, err := getProcessFromParams(params)
	if err != nil {
		return err.Error(), true
	}
	if process.Stage != StageTested {
		return fmt.Sprintf("Invalid state: expected %s but got %s", StageTested, process.Stage), true
	}

	toolObjs := make([]*tool.Tool, 0, len(process.Tools))
	for _, genTool := range process.Tools {
		toolObjs = append(toolObjs, createToolObject(genTool.ToolID, genTool.Description, genTool.InputSchema))
	}

	if err := AddToRegistry(ag.Registry, process.ServerID, process.Description, process.Port, toolObjs); err != nil {
		return fmt.Sprintf("Failed to register server: %v", err), true
	}

	cmd, err := startServerProcess(process.BinaryPath)
	if err != nil {
		return fmt.Sprintf("Failed to start server: %v", err), true
	}

	toolMap := make(map[string]*tool.Tool)
	for _, toolObj := range toolObjs {
		toolMap[toolObj.ToolID] = toolObj
	}

	manager.mu.Lock()
	manager.servers[process.ServerID] = &GeneratedServer{
		ServerID:   process.ServerID,
		Port:       process.Port,
		FilePath:   process.FilePath,
		BinaryPath: process.BinaryPath,
		Process:    cmd,
		Tools:      toolMap,
		Running:    true,
		CreatedAt:  time.Now(),
	}
	process.Stage = StageDeployed
	manager.mu.Unlock()

	toolNames := make([]string, 0, len(process.Tools))
	for _, t := range process.Tools {
		toolNames = append(toolNames, t.ToolID)
	}

	return fmt.Sprintf(
		"SUCCESS! Server '%s' deployed.\n\n"+
			"- Tools: %s\n"+
			"- Port: %d\n"+
			"- Status: Running\n"+
			"- File: %s\n\n"+
			"Next: call cleanup_server_generation with the process_id to remove temporary files.",
		process.ServerID, strings.Join(toolNames, ", "), process.Port, process.FilePath,
	), false
}

// CleanupServerGenerationTool removes temporary artifacts for a process.
func CleanupServerGenerationTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	process, err := getProcessFromParams(params)
	if err != nil {
		return err.Error(), true
	}

	manager.mu.Lock()
	if process.Stage == StageDeployed {
		os.Remove(process.FilePath)
		process.Stage = StageCleaned
		delete(manager.processes, process.ID)
		manager.mu.Unlock()
		return fmt.Sprintf("Cleanup complete for process %s (source removed, binary retained).", process.ID), false
	}
	cleanupArtifacts(process.FilePath, process.BinaryPath)
	port := process.Port
	process.Stage = StageCleaned
	delete(manager.processes, process.ID)
	manager.mu.Unlock()

	manager.deallocatePort(port)
	return fmt.Sprintf("Cleanup complete for process %s (source and binary removed).", process.ID), false
}

// DeleteServerTool removes a generated server and unregisters it.
func DeleteServerTool(ag *agent.Agent, params map[string]interface{}) (string, bool) {
	serverID, ok := params["server_id"].(string)
	if !ok || strings.TrimSpace(serverID) == "" {
		return "delete_server_tool requires 'server_id' parameter", true
	}

	manager.mu.Lock()
	genServer, exists := manager.servers[serverID]
	if !exists {
		manager.mu.Unlock()
		return fmt.Sprintf("server '%s' not found", serverID), true
	}
	delete(manager.servers, serverID)
	manager.mu.Unlock()

	if genServer.Process != nil && genServer.Process.Process != nil {
		genServer.Process.Process.Kill()
	}
	cleanupArtifacts(genServer.FilePath, genServer.BinaryPath)
	manager.deallocatePort(genServer.Port)

	if err := ag.Registry.RemoveServer(serverID); err != nil {
		return fmt.Sprintf("Server '%s' stopped but registry removal failed: %v", serverID, err), true
	}

	return fmt.Sprintf("Server '%s' deleted successfully", serverID), false
}

// ValidateSyntax validates Go code by attempting to compile it.
func ValidateSyntax(sourcePath, binaryPath string) (string, error) {
	cmd := exec.Command("go", "build", "-o", binaryPath, sourcePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("compilation failed")
	}
	return "Syntax validation passed", nil
}

func parseGenerateParams(params map[string]interface{}) (string, string, []GeneratedTool, []string, error) {
	serverID, _ := params["server_id"].(string)
	serverDescription, _ := params["server_description"].(string)
	if strings.TrimSpace(serverID) == "" {
		return "", "", nil, nil, fmt.Errorf("server_id is required")
	}
	if strings.TrimSpace(serverDescription) == "" {
		return "", "", nil, nil, fmt.Errorf("server_description is required")
	}

	toolsRaw, ok := params["tools"].([]interface{})
	if !ok || len(toolsRaw) == 0 {
		return "", "", nil, nil, fmt.Errorf("tools array is required and must be non-empty")
	}

	// Parse optional imports - defaults to empty slice
	imports := make([]string, 0)
	if importsRaw, ok := params["imports"].([]interface{}); ok {
		for _, imp := range importsRaw {
			if impStr, ok := imp.(string); ok && strings.TrimSpace(impStr) != "" {
				imports = append(imports, impStr)
			}
		}
	}

	tools := make([]GeneratedTool, 0, len(toolsRaw))
	for _, t := range toolsRaw {
		toolMap, ok := t.(map[string]interface{})
		if !ok {
			return "", "", nil, nil, fmt.Errorf("each tool must be an object")
		}
		toolID, _ := toolMap["tool_id"].(string)
		description, _ := toolMap["description"].(string)
		inputSchema, _ := toolMap["input_schema"].(map[string]interface{})
		handlerCode, _ := toolMap["handler_code"].(string)
		if strings.TrimSpace(toolID) == "" || strings.TrimSpace(handlerCode) == "" {
			return "", "", nil, nil, fmt.Errorf("each tool must include tool_id and handler_code")
		}
		tools = append(tools, GeneratedTool{
			ToolID:      toolID,
			Description: description,
			InputSchema: inputSchema,
			HandlerCode: handlerCode,
		})
	}

	return serverID, serverDescription, tools, imports, nil
}

func parseToolTests(params map[string]interface{}) (map[string]map[string]interface{}, error) {
	testsRaw, ok := params["tool_tests"].([]interface{})
	if !ok || len(testsRaw) == 0 {
		return nil, fmt.Errorf("tool_tests array is required and must be non-empty")
	}

	tests := make(map[string]map[string]interface{})
	for _, t := range testsRaw {
		toolMap, ok := t.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("each tool_tests entry must be an object")
		}
		toolID, _ := toolMap["tool_id"].(string)
		paramsMap, _ := toolMap["test_params"].(map[string]interface{})
		if strings.TrimSpace(toolID) == "" {
			return nil, fmt.Errorf("tool_tests entries must include tool_id")
		}
		if paramsMap == nil {
			return nil, fmt.Errorf("tool_tests entry for '%s' must include test_params", toolID)
		}
		tests[toolID] = paramsMap
	}

	return tests, nil
}

func assignTestParams(tools []GeneratedTool, tests map[string]map[string]interface{}) []string {
	missing := []string{}
	for i := range tools {
		params, ok := tests[tools[i].ToolID]
		if !ok {
			missing = append(missing, tools[i].ToolID)
			continue
		}
		tools[i].TestParams = params
	}
	return missing
}

func getProcessFromParams(params map[string]interface{}) (*ServerGenerationProcess, error) {
	processID, _ := params["process_id"].(string)
	if strings.TrimSpace(processID) == "" {
		return nil, fmt.Errorf("process_id is required")
	}

	manager.mu.RLock()
	process, exists := manager.processes[processID]
	manager.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("process '%s' not found", processID)
	}

	return process, nil
}

func buildServerSource(serverID string, port int, tools []GeneratedTool, imports []string) string {
	var sb strings.Builder
	sb.WriteString("package main\n\n")

	sb.WriteString("import (\n")

	for _, i := range imports {
		sb.WriteString("\t\"" + i + "\"\n")
	}

	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"log\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("func main() {\n")
	sb.WriteString("\thttp.HandleFunc(\"/execute/\", handleHealth)\n")
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("\thttp.HandleFunc(\"/execute/%s\", handle%s)\n", tool.ToolID, toPascalCase(tool.ToolID)))
	}
	sb.WriteString(fmt.Sprintf("\n\tlog.Printf(\"Starting %s on port %d\")\n", serverID, port))
	sb.WriteString(fmt.Sprintf("\tlog.Fatal(http.ListenAndServe(\":%d\", nil))\n", port))
	sb.WriteString("}\n\n")

	sb.WriteString("func handleHealth(w http.ResponseWriter, r *http.Request) {\n")
	sb.WriteString("\tif r.Method != \"GET\" {\n")
	sb.WriteString("\t\thttp.Error(w, \"Method not allowed\", http.StatusMethodNotAllowed)\n")
	sb.WriteString("\t\treturn\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tw.WriteHeader(http.StatusOK)\n")
	sb.WriteString("\tw.Write([]byte(\"ok\"))\n")
	sb.WriteString("}\n\n")

	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("func handle%s(w http.ResponseWriter, r *http.Request) {\n", toPascalCase(tool.ToolID)))
		sb.WriteString("\tif r.Method != \"POST\" {\n")
		sb.WriteString("\t\thttp.Error(w, \"Method not allowed\", http.StatusMethodNotAllowed)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString(indentCode(tool.HandlerCode, 1))
		sb.WriteString("\n}\n\n")
	}

	return sb.String()
}

func waitForServer(port int, maxRetries int, delay time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/execute/", port)
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	return fmt.Errorf("health check failed: %v", lastErr)
}

func runToolTests(port int, tools []GeneratedTool) (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	var testResults strings.Builder
	testResults.WriteString("TESTING TOOLS:\n")

	for _, tool := range tools {
		url := fmt.Sprintf("http://localhost:%d/execute/%s", port, tool.ToolID)
		jsonData, err := json.Marshal(tool.TestParams)
		if err != nil {
			return fmt.Sprintf("Failed to marshal test payload for %s: %v", tool.ToolID, err), err
		}

		resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Sprintf("Tool '%s': Failed to connect: %v", tool.ToolID, err), err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Sprintf("Tool '%s': Failed to read response: %v", tool.ToolID, err), err
		}
		if resp.StatusCode >= 400 {
			return fmt.Sprintf("Tool '%s': HTTP error %d: %s", tool.ToolID, resp.StatusCode, string(body)), fmt.Errorf("http error")
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Sprintf("Tool '%s': Returned invalid JSON: %s", tool.ToolID, string(body)), err
		}

		testResults.WriteString(fmt.Sprintf("\nOK %s\n", tool.ToolID))
		testResults.WriteString(fmt.Sprintf("  Response: %s\n", string(body)))
	}

	return testResults.String(), nil
}

// AddToRegistry registers a generated server in the agent's registry.
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

func startServerProcess(binaryPath string) (*exec.Cmd, error) {
	cmd := exec.Command(binaryPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	log.Printf("Started server (PID: %d)", cmd.Process.Pid)
	return cmd, nil
}

func stopProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	cmd.Process.Kill()
}

func cleanupProcess(process *ServerGenerationProcess) {
	cleanupArtifacts(process.FilePath, process.BinaryPath)
	manager.deallocatePort(process.Port)
	manager.mu.Lock()
	delete(manager.processes, process.ID)
	process.Stage = StageFailed
	manager.mu.Unlock()
}

func cleanupArtifacts(filePath, binaryPath string) {
	os.Remove(filePath)
	os.Remove(binaryPath)
}

// resolveBinaryPath ensures the correct executable path for the current OS.
func resolveBinaryPath(basePath string) string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(basePath), ".exe") {
		return basePath + ".exe"
	}
	return basePath
}

// createToolObject creates a Tool object for registration.
func createToolObject(toolID, description string, inputSchema map[string]interface{}) *tool.Tool {
	properties, _ := inputSchema["properties"].(map[string]interface{})
	required, _ := inputSchema["required"].([]interface{})

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

// allocatePort returns the next available port.
func (sm *ServerManager) allocatePort() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	port := sm.nextPort
	sm.nextPort++
	return port
}

// deallocatePort marks a port as available for reuse.
func (sm *ServerManager) deallocatePort(port int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if port == sm.nextPort-1 {
		sm.nextPort--
	}
}

func (sm *ServerManager) newProcessID() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.nextProcessID++
	return fmt.Sprintf("proc_%d", sm.nextProcessID)
}

// toPascalCase converts snake_case to PascalCase.
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// indentCode adds indentation to code.
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
