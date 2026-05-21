Phase 1 — Setup (Week 1) — Zane + Anthony

Create /security/ folder in GOMCP repo with its own README.md explaining the research scope, demo instructions, and toggle mechanism
Add VULN_MODE flag: Go API reads an env var (VULN_MODE=true), React reads VITE_VULN_MODE; when true, security mitigations are bypassed/skipped
Introduce the 4 deliberate vulnerabilities into the codebase under VULN_MODE guards:
Zane: Add dangerouslySetInnerHTML path in ChatPage.tsx (XSS); remove Content-Security-Policy and CORS restriction headers in Go API
Anthony: Remove JWT middleware from Go API routes; set Cognito user pool to allow unlimited login retries
Deploy the app to AWS EC2/Amplify with VULN_MODE=true first so both can reproduce attacks
Phase 2 — XSS + Broken Access Control (Week 2) — parallel
5. Zane – XSS: Pen-test with a crafted payload in the chat input (<script>alert(1)</script>, cookie stealer). Document with screenshots. Then harden: remove dangerouslySetInnerHTML, add DOMPurify sanitization, add Content-Security-Policy header. Write security/how-to/01-xss.md
6. Anthony – Broken Access Control: Pen-test by calling /api/chat/message without a JWT token (raw curl). Document. Then harden: add Go middleware that validates Cognito JWT on every /api/ route. Write security/how-to/02-broken-access-control.md

Phase 3 — Security Misconfiguration + Broken Auth (Week 3) — parallel
7. Zane – Security Misconfiguration: Pen-test using browser dev tools + curl -I to show missing headers, overly permissive CORS. Document. Then harden: set Access-Control-Allow-Origin to the deployed frontend domain, add X-Frame-Options: DENY, X-Content-Type-Options: nosniff, Strict-Transport-Security, Content-Security-Policy in Go middleware. Suppress raw error strings in responses. Write security/how-to/03-security-misconfiguration.md
8. Anthony – Broken Authentication: Pen-test with Hydra or a custom script to brute-force the Cognito login. Document. Then harden: enable Cognito account lockout / advanced security features, enforce rate limiting on /login via the Go API proxy, validate JWT expiry and signature. Write security/how-to/04-broken-authentication.md

Phase 4 — Polish + Demo Prep (Week 4)
9. Each write-up should include: Background, Attack steps (with commands/screenshots), Mitigation code diff, Verification test, References
10. Add security/scripts/ with named shell/Python scripts for each attack so results are fully reproducible
11. Update top-level security/README.md with the toggle instructions (VULN_MODE=true/false) and a quick-start for the grader

Phase 5 — Buffer / Dry Run (Week 5)
12. Run full demo end-to-end; verify VULN_MODE=false hardens all 4 attacks simultaneously
13. Fix any issues found in dry run

Relevant files to modify
ChatPage.tsx — XSS surface (response rendering)
api.ts — frontend API calls, JWT attachment
AuthGuard.tsx — client-side auth guard (not a substitute for server-side)
Go API handler files (to be created in Sprint 2, or existing server.go) — CORS, security headers, JWT middleware
AWS Cognito User Pool settings — advanced security / rate limiting
New: security/ folder structure

Verification
VULN_MODE=true: Each of the 4 attacks succeeds using the scripts in security/scripts/
VULN_MODE=false: Each attack is blocked/returns a 403 or sanitized output
A grader with no prior knowledge can follow each how-to/*.md and reproduce both the attack and the fix from scratch
