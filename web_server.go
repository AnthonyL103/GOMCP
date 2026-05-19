package main

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AnthonyL103/GOMCP/store"
)

// ============================================================================
// Cognito JWT validation — no third-party JWT library required
// ============================================================================

// cognitoRegion and cognitoUserPoolID are set in main() from env vars.
var (
	cognitoRegion     string
	cognitoUserPoolID string
	cognitoJWKS       jwksCache
)

type jwksCache struct {
	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey // kid -> RSA public key
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// fetchFromCognito downloads the JWKS for the configured user pool and caches
// the RSA public keys in memory. Call once at startup.
func (c *jwksCache) fetchFromCognito() error {
	url := fmt.Sprintf(
		"https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json",
		cognitoRegion, cognitoUserPoolID,
	)

	resp, err := http.Get(url) //nolint:noctx — one-off startup call
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Keys []jwkKey `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	parsed := make(map[string]*rsa.PublicKey, len(result.Keys))
	for _, k := range result.Keys {
		if k.Use != "sig" || k.Kty != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		e := 0
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}
		parsed[k.Kid] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: e,
		}
	}

	c.mu.Lock()
	c.keys = parsed
	c.mu.Unlock()
	return nil
}

func (c *jwksCache) getKey(kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	key, ok := c.keys[kid]
	c.mu.RUnlock()
	if ok {
		return key, nil
	}
	return nil, fmt.Errorf("key %q not found in JWKS cache", kid)
}

// verifyJWT validates a Cognito-issued RS256 JWT.
// Returns the Cognito sub (user ID) on success, or an error.
func verifyJWT(tokenStr string) (string, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return "", errors.New("malformed token")
	}

	// Decode header to get key ID and algorithm.
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", fmt.Errorf("parse header: %w", err)
	}
	if header.Alg != "RS256" {
		return "", fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	// Look up the public key by kid.
	pubKey, err := cognitoJWKS.getKey(header.Kid)
	if err != nil {
		return "", err
	}

	// Verify the signature: RS256 = RSASSA-PKCS1-v1_5 with SHA-256.
	signingInput := parts[0] + "." + parts[1]
	hash := sha256.Sum256([]byte(signingInput))
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("decode signature: %w", err)
	}
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes); err != nil {
		return "", fmt.Errorf("invalid signature: %w", err)
	}

	// Decode payload and validate standard claims.
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode payload: %w", err)
	}
	var claims struct {
		Sub string  `json:"sub"`
		Exp float64 `json:"exp"`
		Iss string  `json:"iss"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return "", fmt.Errorf("parse claims: %w", err)
	}
	if time.Now().Unix() > int64(claims.Exp) {
		return "", errors.New("token expired")
	}
	expectedIss := fmt.Sprintf(
		"https://cognito-idp.%s.amazonaws.com/%s",
		cognitoRegion, cognitoUserPoolID,
	)
	if claims.Iss != expectedIss {
		return "", errors.New("invalid token issuer")
	}

	return claims.Sub, nil
}

// ============================================================================
// Middleware
// ============================================================================

type contextKey string

const contextKeyUserID contextKey = "userID"

// corsMiddleware adds permissive CORS headers and handles OPTIONS preflight.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
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

// jwtMiddleware validates the Bearer token from the Authorization header.
// On success it stores the user's Cognito sub in the request context.
func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "missing or invalid Authorization header",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := verifyJWT(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "invalid or expired token",
			})
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ============================================================================
// Handler struct
// ============================================================================

type handler struct {
	store *store.ProjectStore
	// provider and agent will be wired in Phase 2 for LLM generation.
}

// ============================================================================
// Route handlers — stubs filled out in Phase 2
// ============================================================================

func (h *handler) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	// Phase 2: accept answers JSON, create project, kick off generation goroutine.
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (h *handler) handleListProjects(w http.ResponseWriter, r *http.Request) {
	// Phase 2: return all projects for the authenticated user.
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (h *handler) handleGetProject(w http.ResponseWriter, r *http.Request) {
	// Phase 2: return a single project by ID.
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (h *handler) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	// Phase 2: append a user message and trigger the next LLM turn.
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (h *handler) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	// Phase 2: return the saved Terraform script for a project.
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (h *handler) handleDeploy(w http.ResponseWriter, r *http.Request) {
	// Phase 2: run terraform plan against the saved script.
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (h *handler) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	// Phase 2: return the most recent deployment result.
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// ============================================================================
// Helpers
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ============================================================================
// Server startup
// ============================================================================

func startWebServer(ps *store.ProjectStore) {
	h := &handler{store: ps}

	// Inner mux holds the actual routes.
	inner := http.NewServeMux()
	inner.HandleFunc("POST /projects", h.handleCreateProject)
	inner.HandleFunc("GET /projects", h.handleListProjects)
	inner.HandleFunc("GET /projects/{id}", h.handleGetProject)
	inner.HandleFunc("POST /projects/{id}/messages", h.handleCreateMessage)
	inner.HandleFunc("GET /projects/{id}/artifact", h.handleGetArtifact)
	inner.HandleFunc("POST /projects/{id}/deploy", h.handleDeploy)
	inner.HandleFunc("GET /projects/{id}/deployment", h.handleGetDeployment)

	// Wrap all routes with JWT auth, then CORS on the outside.
	root := http.NewServeMux()
	root.Handle("/", jwtMiddleware(inner))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(root),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second, // generous for LLM-backed endpoints
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("web server listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}
