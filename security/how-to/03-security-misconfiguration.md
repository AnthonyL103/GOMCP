# 03 — Security Misconfiguration

**OWASP Category:** A05 Security Misconfiguration  
**Owner:** Zane  
**Status:** Implemented (Phase 3)

---

## Background

Security Misconfiguration is the most common finding in real-world audits. It covers a broad class of issues: missing HTTP security headers, permissive CORS policies, verbose error messages, default credentials, and unnecessary features left enabled. GOMCP demonstrates two specific sub-issues:

1. **Permissive CORS** — the server reflects any `Origin` header back, allowing requests from any domain.
2. **Missing security headers** — `Content-Security-Policy`, `X-Frame-Options`, `X-Content-Type-Options`, `Strict-Transport-Security`, and `Referrer-Policy` are absent.

**Impact:** Missing headers enable clickjacking, MIME-sniffing attacks, open-redirect chains, and weaken XSS defences. Permissive CORS allows attacker-controlled pages to make credentialed API requests on behalf of authenticated users.

---

## Vulnerability Location

**File:** `web_server.go` — `corsMiddleware` and `securityHeadersMiddleware`  
**Condition:** `VULN_MODE=true`

```go
// VULNERABLE: reflects any Origin back
if origin := r.Header.Get("Origin"); origin != "" {
    w.Header().Set("Access-Control-Allow-Origin", origin)
}
// VULNERABLE: no security headers set
```

---

## Attack Steps

### Prerequisites

1. Start the backend in vulnerable mode:
   ```bash
   VULN_MODE=true go run .
   ```

### Attack 1 — Inspect missing headers

```bash
curl -si http://localhost:8080/projects -X OPTIONS \
  -H "Origin: https://evil.example.com" \
  -H "Access-Control-Request-Method: POST" | head -30
```

**Expected result (VULN_MODE=true):**
- `Access-Control-Allow-Origin: https://evil.example.com` — server echoes attacker origin
- No `Content-Security-Policy` header
- No `X-Frame-Options` header
- No `X-Content-Type-Options` header
- No `Strict-Transport-Security` header

### Attack 2 — Clickjacking via missing X-Frame-Options

Without `X-Frame-Options: DENY` any page can embed the app in an `<iframe>`:

```html
<!-- attacker-controlled page -->
<iframe src="https://your-app.example.com" style="opacity:0; position:absolute; top:0; left:0; width:100%; height:100%"></iframe>
<button onclick="doSomething()">Click to win a prize!</button>
```

A victim who clicks the invisible button actually clicks a button inside the authenticated app.

### Attack 3 — Cross-origin credentialed request (CORS)

```bash
# Simulate a request from an attacker-controlled origin
curl -si http://localhost:8080/projects \
  -H "Origin: https://attacker.evil" \
  -H "Authorization: Bearer VICTIM_TOKEN" | grep -i "access-control"
```

**Expected result (VULN_MODE=true):** `Access-Control-Allow-Origin: https://attacker.evil`

A browser-based attack page at `https://attacker.evil` can make credentialed `fetch()` calls to the API and read the responses.

### Automation Script

```bash
bash security/scripts/03-misconfiguration-attack.sh
```

---

## Mitigation

### 1 — Restrict CORS to the deployed frontend origin

Set `ALLOWED_ORIGIN` to your actual frontend domain. In `web_server.go`:

```go
// HARDENED: only the configured origin is allowed
allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
if allowedOrigin == "" {
    allowedOrigin = "http://localhost:5173"
}
w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
```

### 2 — Add security headers

Applied in `securityHeadersMiddleware` when `VULN_MODE=false`:

| Header | Value | Protects Against |
|--------|-------|-----------------|
| `Content-Security-Policy` | `default-src 'self'; script-src 'self'; object-src 'none'; frame-ancestors 'none'` | XSS, clickjacking |
| `X-Frame-Options` | `DENY` | Clickjacking |
| `X-Content-Type-Options` | `nosniff` | MIME-type confusion |
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains; preload` | SSL stripping |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Referrer leakage |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` | Feature abuse |

### 3 — Suppress internal error details

Never return raw Go error strings in API responses. Use generic messages:

```go
// BAD
writeJSON(w, 500, map[string]string{"error": err.Error()})

// GOOD
log.Printf("internal error: %v", err)
writeJSON(w, 500, map[string]string{"error": "internal server error"})
```

---

## Verification Test

### VULN_MODE=true (misconfiguration visible)

```bash
curl -si http://localhost:8080/projects -X OPTIONS \
  -H "Origin: https://evil.example.com" | grep -E "access-control-allow-origin|content-security-policy|x-frame-options"
# Expected: Access-Control-Allow-Origin: https://evil.example.com
# Expected: no CSP or X-Frame-Options lines
```

### VULN_MODE=false (hardened)

```bash
export ALLOWED_ORIGIN=http://localhost:5173
curl -si http://localhost:8080/projects -X OPTIONS \
  -H "Origin: https://evil.example.com" | grep -E "access-control-allow-origin|content-security-policy|x-frame-options"
# Expected: Access-Control-Allow-Origin: http://localhost:5173  (attacker origin NOT echoed back)
# Expected: Content-Security-Policy header present
# Expected: X-Frame-Options: DENY
```

---

## References

- [OWASP Security Misconfiguration](https://owasp.org/Top10/A05_2021-Security_Misconfiguration/)
- [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/)
- [MDN: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [MDN: X-Frame-Options](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Frame-Options)
- [MDN: Content-Security-Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
