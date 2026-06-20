package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	agent "github.com/AnthonyL103/GOMCP/Agent"
	"github.com/AnthonyL103/GOMCP/chat"
	"github.com/AnthonyL103/GOMCP/store"
	"github.com/AnthonyL103/GOMCP/transport"
	"github.com/gorilla/websocket"
)

// -------------------------------------------------------------------
// WebSocket hub
// -------------------------------------------------------------------

type Hub struct {
	mu      sync.RWMutex
	writeMu sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func newHub() *Hub { return &Hub{clients: make(map[*websocket.Conn]struct{})} }

func (h *Hub) add(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	c.Close()
}

// Broadcast sends any value as JSON to all connected WS clients.
func (h *Hub) Broadcast(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("[Hub] Broadcast marshal error: %v", err)
		return
	}
	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	clientCount := len(clients)
	h.mu.RUnlock()
	log.Printf("[Hub] Broadcasting to %d clients: %s", clientCount, string(data[:min(len(data), 200)]))

	// Serialize writes because gorilla/websocket connections are not safe for concurrent writers.
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	for _, c := range clients {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[Hub] Write error: %v", err)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// -------------------------------------------------------------------
// Server
// -------------------------------------------------------------------

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	ag       *agent.Agent
	provider transport.Provider
	chat     *chat.Chat
	hub      *Hub
	store    *store.ProjectStore
	sessions *SessionManager
}

// POST /chat — same logic as the CLI loop, just over HTTP
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
		UserID    string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if body.SessionID == "" {
		body.SessionID = "session-1"
	}

	userEmail := strings.TrimSpace(body.UserID)
	if userEmail != "" {
		log.Printf("Authenticated Cognito user email for chat request: %s", userEmail)
	}

	c := s.loadStoredChat(body.SessionID)
	continuationContext := ""
	if proj, ok := s.store.Get(body.SessionID); ok && len(proj.Messages) > 0 {
		continuationContext = buildContinuationContext(proj)
	}

	if err := s.provider.SendRequest(c, s.ag, body.Message, userEmail, continuationContext); err != nil {
		log.Printf("Error: %v", err)
		s.hub.Broadcast(map[string]string{"type": "error", "message": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messages := c.GetMessages()
	log.Printf("Chat request complete for session %s, total messages: %d", body.SessionID, len(messages))

	if len(messages) == 0 {
		http.Error(w, "no response", http.StatusInternalServerError)
		return
	}

	lastMsg := messages[len(messages)-1]

	// Persist chat and extract terraform script as needed.
	go s.persistChat(body.SessionID, userEmail, c)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lastMsg)
}

// persistChat synchronises the in-memory chat with the ProjectStore and
// extracts any Terraform script present in the conversation.
func (s *Server) persistChat(sessionID, userID string, c *chat.Chat) {
	if s.store == nil {
		return
	}

	// Load existing project if present
	proj, ok := s.store.Get(sessionID)
	isNew := !ok
	if isNew {
		proj = &store.Project{
			ID:        sessionID,
			UserID:    userID,
			Name:      deriveName(c, sessionID),
			Status:    store.StatusGenerating,
			CreatedAt: time.Now(),
		}
	}
	if userID != "" {
		proj.UserID = userID
	}

	proj.Messages = toStoreMessages(c.GetMessages())

	script := extractTerraformScript(c.GetMessages())
	hadScript := proj.TerraformScript != ""
	if script != "" {
		proj.TerraformScript = script
		proj.Status = store.StatusComplete
	}

	if isNew {
		proj.CreatedAt = time.Now()
		proj.UpdatedAt = time.Now()
		_ = s.store.Create(proj)
	} else {
		proj.UpdatedAt = time.Now()
		_ = s.store.Update(proj)
	}

	if script != "" {
		_ = os.WriteFile(s.store.ArtifactPath(sessionID), []byte(script), 0644)
	}

	if script != "" && !hadScript {
		if s.hub != nil {
			s.hub.Broadcast(map[string]string{"type": "script_ready", "session_id": sessionID, "project_id": sessionID})
		}
	}
}

func toStoreMessages(msgs []chat.Message) []store.ChatMessage {
	out := make([]store.ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		var toolCall *store.ToolCall
		if m.ToolCall != nil {
			toolCall = &store.ToolCall{
				ServerID:   m.ToolCall.ServerID,
				ToolID:     m.ToolCall.ToolID,
				Handler:    m.ToolCall.Handler,
				Parameters: m.ToolCall.Parameters,
				Reasoning:  m.ToolCall.Reasoning,
				ToolUseID:  m.ToolCall.ToolUseID,
			}
		}

		var toolResult *store.ToolResult
		if m.ToolResult != nil {
			toolResult = &store.ToolResult{
				ServerID:  m.ToolResult.ServerID,
				ToolID:    m.ToolResult.ToolID,
				Content:   m.ToolResult.Content,
				IsError:   m.ToolResult.IsError,
				ToolUseID: m.ToolResult.ToolUseID,
			}
		}

		out = append(out, store.ChatMessage{
			Role:       m.Role,
			Content:    m.Content,
			CreatedAt:  m.Timestamp,
			ToolCall:   toolCall,
			ToolResult: toolResult,
		})
	}
	return out
}

func buildContinuationContext(proj *store.Project) string {
	if proj == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("This chat is a continuation of an existing saved project.")
	if strings.TrimSpace(proj.Name) != "" {
		sb.WriteString("\nProject name: ")
		sb.WriteString(proj.Name)
	}
	if strings.TrimSpace(proj.ID) != "" {
		sb.WriteString("\nProject ID: ")
		sb.WriteString(proj.ID)
	}
	sb.WriteString("\nProject status: ")
	sb.WriteString(string(proj.Status))

	if len(proj.Messages) > 0 {
		sb.WriteString("\n\nRecent chat transcript:")
		start := 0
		if len(proj.Messages) > 16 {
			start = len(proj.Messages) - 16
		}
		for _, msg := range proj.Messages[start:] {
			content := strings.TrimSpace(msg.Content)
			if content == "" {
				continue
			}
			sb.WriteString("\n")
			if msg.Role == "assistant" {
				sb.WriteString("Assistant: ")
			} else {
				sb.WriteString("User: ")
			}
			sb.WriteString(content)
		}
	}

	if strings.TrimSpace(proj.TerraformScript) != "" {
		sb.WriteString("\n\nCurrent Terraform script:\n")
		sb.WriteString(proj.TerraformScript)
	}

	return sb.String()
}

func deriveName(c *chat.Chat, sessionID string) string {
	msgs := c.GetMessages()
	for _, m := range msgs {
		if m.Role == "user" {
			// look for a line like "Project name: NAME"
			lines := strings.Split(m.Content, "\n")
			for _, l := range lines {
				if strings.Contains(strings.ToLower(l), "project name:") {
					parts := strings.SplitN(l, ":", 2)
					if len(parts) == 2 {
						name := strings.TrimSpace(parts[1])
						if name != "" {
							return name
						}
					}
				}
			}
		}
	}
	if len(sessionID) >= 8 {
		return "Project " + sessionID[:8]
	}
	return "Project " + sessionID
}

var fencedRe = regexp.MustCompile("(?s)```(?:hcl|terraform|tf)\\n(.*?)```")

func extractTerraformScript(msgs []chat.Message) string {
	// newest-first
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.ToolCall != nil {
			if m.ToolCall.ToolID == "save_project_session" {
				if v, ok := m.ToolCall.Parameters["terraform_script"].(string); ok && strings.TrimSpace(v) != "" {
					return v
				}
			}
			if m.ToolCall.ToolID == "generate_aws_terraform_iteration" {
				if v, ok := m.ToolCall.Parameters["terraform"].(string); ok && strings.TrimSpace(v) != "" {
					return v
				}
			}
		}
		if m.ToolResult != nil {
			if strings.TrimSpace(m.ToolResult.Content) != "" {
				if script := extractAllFencedTerraformBlocks(m.ToolResult.Content); strings.TrimSpace(script) != "" {
					return script
				}
			}
		}
		// assistant's content
		if m.Role == "assistant" {
			if script := extractAllFencedTerraformBlocks(m.Content); strings.TrimSpace(script) != "" {
				return script
			}
		}
	}
	return ""
}

func extractAllFencedTerraformBlocks(content string) string {
	matches := fencedRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return ""
	}

	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		block := strings.TrimSpace(match[1])
		if block == "" {
			continue
		}
		parts = append(parts, block)
	}

	return strings.Join(parts, "\n\n")
}

// GET /ws — clients connect here to receive tool update broadcasts
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] upgrade error: %v", err)
		return
	}
	s.hub.add(conn)
	log.Printf("[WS] client connected: %s, total clients: %d", conn.RemoteAddr(), len(s.hub.clients))

	go func() {
		defer func() {
			s.hub.remove(conn)
			log.Printf("[WS] client disconnected: %s", conn.RemoteAddr())
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				if err.Error() != "websocket: close sent" {
					log.Printf("[WS] read error: %v", err)
				}
				break
			}
		}
	}()
}

// POST /done — trigger cleanup of cloned repos and clear chat context
func (s *Server) handleDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("Cleanup requested via /done endpoint")

	// Clear chat context for next audit
	s.chat = chat.NewChat("session-1", 50)
	log.Println("Chat context cleared")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "cleaned"})
}

// -------------------------------------------------------------------
// Unchanged helpers
// -------------------------------------------------------------------

func findModel(model string) string {
	for _, m := range []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "o1-preview", "o1-mini"} {
		if m == model {
			return "OpenAI"
		}
	}
	for _, m := range []string{"claude-opus-4-5-20251101", "claude-sonnet-4-5-20250929", "claude-haiku-4-5-20251001"} {
		if m == model {
			return "Anthropic"
		}
	}
	return ""
}

func createProvider(ag *agent.Agent) (transport.Provider, error) {
	switch findModel(ag.LLMConfig.Model) {
	case "Anthropic":
		return transport.NewAnthropicProvider(ag.LLMConfig), nil
	case "OpenAI":
		return transport.NewOpenAIProvider(ag.LLMConfig), nil
	default:
		return nil, fmt.Errorf("unsupported model: %s", ag.LLMConfig.Model)
	}
}

func setupGracefulShutdown(processes []*os.Process) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		for _, proc := range processes {
			if proc != nil {
				proc.Kill()
			}
		}

		os.Exit(0)
	}()
}
