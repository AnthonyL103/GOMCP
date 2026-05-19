package main

import (
	"log"
	"os"

	"github.com/AnthonyL103/GOMCP/store"
)

func main() {
	// Load Cognito config from environment variables.
	cognitoRegion = os.Getenv("COGNITO_REGION")
	cognitoUserPoolID = os.Getenv("COGNITO_USER_POOL_ID")
	if cognitoRegion == "" || cognitoUserPoolID == "" {
		log.Fatal("COGNITO_REGION and COGNITO_USER_POOL_ID environment variables are required")
	}

	// Fetch and cache Cognito's public keys so every request can be verified.
	if err := cognitoJWKS.fetchFromCognito(); err != nil {
		log.Fatalf("failed to load Cognito JWKS: %v", err)
	}
	log.Println("Cognito JWKS loaded")

	// Initialise the project store (creates ./data directories on first run).
	ps, err := store.NewProjectStore("./data")
	if err != nil {
		log.Fatalf("failed to init project store: %v", err)
	}
	log.Println("project store ready")

	// Start the HTTP server (blocking).
	startWebServer(ps)
}
