package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	agent "github.com/AnthonyL103/GOMCP/Agent"
	"github.com/AnthonyL103/GOMCP/chat"
	"github.com/AnthonyL103/GOMCP/protocol/parseagentprotocol"
	"github.com/AnthonyL103/GOMCP/transport"
	voicechat "github.com/AnthonyL103/GOMCP/voice"
	"github.com/joho/godotenv"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:5173"
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func runConsoleChat(ag *agent.Agent, provider transport.Provider) {
	chatSession := chat.NewChat("session-1", 50)
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Console mode ready. Type a message and press Enter. Type 'exit' to quit.")

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("read input: %v", err)
			return
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
			return
		}

		if err := provider.SendRequest(chatSession, ag, input); err != nil {
			log.Printf("agent error: %v", err)
			continue
		}

		messages := chatSession.GetRecentMessages(1)
		if len(messages) == 0 {
			continue
		}

		last := messages[len(messages)-1]
		if last.Content != "" {
			fmt.Println(last.Content)
		}
	}
}

func main() {

	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file found, using system env")
	} else {
		log.Println("✓ Loaded .env successfully")
	}

	ag, err := parseagentprotocol.ParseAgentConfig()
	if err != nil {
		log.Fatalf("parse agent config: %v", err)
	}

	log.Println("Starting MCP servers...")
	processes, err := StartAllServers(ag)
	if err != nil {
		log.Fatalf("start servers: %v", err)
	}
	setupGracefulShutdown(processes)
	time.Sleep(2 * time.Second)
	log.Println("All servers started!")

	provider, err := createProvider(ag)
	if err != nil {
		log.Fatalf("create provider: %v", err)
	}
	log.Printf("Using provider: %s", provider.GetProviderName())

	if err != nil {
		log.Fatalf("init project store: %v", err)
	}

	if ag.ApiMode {
		hub := newHub()

			// Wire callback after hub exists
			if ap, ok := provider.(*transport.AnthropicProvider); ok {
				log.Println("Setting AnthropicProvider OnToolCall callback to broadcast tool calls to WS clients")
				ap.OnToolCall = func(msg chat.Message) {
					hub.Broadcast(msg)
				}
			}
			if op, ok := provider.(*transport.OpenAIProvider); ok {
				log.Println("Setting OpenAIProvider OnToolCall callback to broadcast tool calls to WS clients")
				op.OnToolCall = func(msg chat.Message) {
					hub.Broadcast(msg)
				}
			}

			srv := &Server{
				ag:       ag,
				provider: provider,
				chat:     chat.NewChat("session-1", 50),
				hub:      hub,
			}

			if ag.VoiceChat {
				vcParser := voicechat.NewVoiceChatParser(srv.chat, ag, provider)
				go vcParser.Start()
			}

			mux := http.NewServeMux()
			mux.HandleFunc("/chat", srv.handleChat)
			mux.HandleFunc("/ws", srv.handleWS)
			mux.HandleFunc("/done", srv.handleDone)

			log.Println("Listening on :8080")
			log.Fatal(http.ListenAndServe(":8080", corsMiddleware(mux)))
	} else {
		runConsoleChat(ag, provider)
	}
	
}

	
