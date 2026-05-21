# GOMCP Security Research

This directory contains a deliberately vulnerable copy of the GOMCP frontend and supporting documentation for a structured security research project. The goal is to demonstrate four OWASP Top-10 vulnerabilities in a controlled environment, then harden against each one.

## Vulnerabilities Covered

| # | Vulnerability | OWASP Category | Owner |
|---|---------------|----------------|-------|
| 1 | Stored/Reflected XSS via chat response rendering | A03 Injection | Zane |
| 2 | Broken Access Control – unauthenticated API calls | A01 Broken Access Control | Anthony |
| 3 | Security Misconfiguration – missing headers / permissive CORS | A05 Security Misconfiguration | Zane |
| 4 | Broken Authentication – Cognito brute-force | A07 Identification & Auth Failures | Anthony |

## VULN_MODE Toggle

All vulnerabilities are guarded by a single flag so the same codebase demonstrates both the attack and the fix.

### Backend (Go)

Set the environment variable before starting the server:

```bash
# Vulnerable — all mitigations disabled
VULN_MODE=true go run .

# Hardened — all mitigations active (default)
VULN_MODE=false go run .
```

### Frontend (React / Vite)

Create `security/web/.env.local` (never committed):

```
# Vulnerable
VITE_VULN_MODE=true

# Hardened
VITE_VULN_MODE=false
```

Or pass it inline:

```bash
VITE_VULN_MODE=true npm run dev
```

## Quick-Start for Graders

```bash
# 1. Start the backend in vulnerable mode
cd /path/to/GOMCP
VULN_MODE=true go run .

# 2. Start the security frontend in vulnerable mode
cd security/web
VITE_VULN_MODE=true npm run dev

# 3. Follow the attack scripts
bash security/scripts/01-xss-attack.sh
bash security/scripts/03-misconfiguration-attack.sh

# 4. Flip the flag and verify each attack is blocked
VULN_MODE=false go run .
# (restart frontend with VITE_VULN_MODE=false)
bash security/scripts/01-xss-attack.sh   # should be blocked / sanitized
bash security/scripts/03-misconfiguration-attack.sh  # headers should be present
```

## Directory Structure

```
security/
├── README.md                  ← you are here
├── plan.md                    ← sprint plan
├── how-to/
│   ├── 01-xss.md              ← XSS background, attack steps, mitigation
│   ├── 02-broken-access-control.md
│   ├── 03-security-misconfiguration.md
│   └── 04-broken-authentication.md
├── scripts/
│   ├── 01-xss-attack.sh
│   ├── 02-bac-attack.sh
│   ├── 03-misconfiguration-attack.sh
│   └── 04-brute-force-attack.sh
└── web/                       ← copy of the React frontend used for demos
```

## Verification Matrix

| Attack | VULN_MODE=true | VULN_MODE=false |
|--------|----------------|-----------------|
| XSS payload `<script>alert(1)</script>` | Alert fires | Sanitized, no execution |
| Unauthenticated `POST /projects` | 200 OK | 401 Unauthorized |
| `curl -I` missing security headers | Headers absent | All headers present |
| Brute-force 100 login attempts | Succeeds | Account locked / rate-limited |
