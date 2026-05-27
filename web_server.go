package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"regexp"
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

// vuln_mode returns true when the VULN_MODE environment variable is set to "true".
// When true, security mitigations are bypassed so attacks can be demonstrated.
func vulnMode() bool {
	return strings.EqualFold(os.Getenv("VULN_MODE"), "true")
}

// securityHeadersMiddleware adds defensive HTTP headers.
//
// VULN_MODE=true  → headers are omitted (Security Misconfiguration demo).
// VULN_MODE=false → full set of security headers is applied.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !vulnMode() {
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; object-src 'none'; frame-ancestors 'none'")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			// Never leak internal error details via the Server header
			w.Header().Set("Server", "")
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
	userID, ok := r.Context().Value(contextKeyUserID).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid user context"})
		return
	}

	projects := h.store.GetByUserID(userID)
	if projects == nil {
		projects = []*store.Project{}
	}

	writeJSON(w, http.StatusOK, projects)
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
// Security-demo chat handlers (no JWT required — XSS demo surface)
// ============================================================================

// htmlTagRe strips HTML tags for the hardened chat response path.
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// handleChatSession issues a random session ID.
// No auth required — intentional for the Broken Access Control demo (vuln #2).
func (h *handler) handleChatSession(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"session_id": hex.EncodeToString(b)})
}

// handleChatMessage echoes the user message back as the "AI response".
//
// VULN_MODE=true  → response contains the raw message (HTML/scripts intact — XSS demo).
// VULN_MODE=false → HTML tags are stripped before the response is written.
func (h *handler) handleChatMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	response := body.Message
	if !vulnMode() {
		// HARDENED: strip all HTML tags before echoing back
		response = htmlTagRe.ReplaceAllString(response, "")
	}

	writeJSON(w, http.StatusOK, map[string]string{"response": response})
}

// ============================================================================
// Helpers
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// webHandler returns the protected project API handler mounted under the root
// mux used by main().
func webHandler(ps *store.ProjectStore) http.Handler {
	h := &handler{store: ps}

	inner := http.NewServeMux()
	inner.HandleFunc("POST /projects", h.handleCreateProject)
	inner.HandleFunc("GET /projects", h.handleListProjects)
	inner.HandleFunc("GET /projects/{id}", h.handleGetProject)
	inner.HandleFunc("POST /projects/{id}/messages", h.handleCreateMessage)
	inner.HandleFunc("GET /projects/{id}/artifact", h.handleGetArtifact)
	inner.HandleFunc("POST /projects/{id}/deploy", h.handleDeploy)
	inner.HandleFunc("GET /projects/{id}/deployment", h.handleGetDeployment)

	// If running in demo/vuln mode, or DISABLE_AUTH=true, or Cognito not configured,
	// expose the project API without JWT validation so local demos work.
	if vulnMode() || strings.EqualFold(os.Getenv("DISABLE_AUTH"), "true") || cognitoRegion == "" || cognitoUserPoolID == "" {
		log.Println("Auth disabled for project API: vuln mode / DISABLE_AUTH / missing Cognito config detected")
		return inner
	}

	return jwtMiddleware(inner)
}
