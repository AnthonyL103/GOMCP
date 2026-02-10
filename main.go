package main

import (
	"log"

	"github.com/AnthonyL103/GOMCP/protocol/parseagentprotocol"
	"github.com/AnthonyL103/GOMCP/protocol/llmprotocol"
)

func main() {
	agent, err := parseagentprotocol.ParseAgentConfig()
	if err != nil {
		log.Fatal(err)
	}

	req, err := llmprotocol.BuildLLMRequest(agent, "What's the weather?")
	if err != nil {
		log.Fatal(err)
	}

	llmprotocol.PrintLLMRequest(req)
}