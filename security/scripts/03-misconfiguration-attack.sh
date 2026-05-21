#!/usr/bin/env bash
# security/scripts/03-misconfiguration-attack.sh
# Demonstrates permissive CORS and missing security headers.
# Run with VULN_MODE=true to see the attack surface.
# Run with VULN_MODE=false (+ ALLOWED_ORIGIN set) to verify hardening.

set -euo pipefail

BASE="${API_BASE:-http://localhost:8080}"
EVIL_ORIGIN="https://attacker.evil"

echo "=== Security Misconfiguration Attack Script ==="
echo "Target: ${BASE}"
echo ""

# Test 1: CORS — attacker origin echoed back?
echo "[1] Testing CORS with attacker-controlled Origin..."
CORS_RESP=$(curl -si -X OPTIONS "${BASE}/projects" \
  -H "Origin: ${EVIL_ORIGIN}" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type, Authorization")

ACAO=$(echo "${CORS_RESP}" | grep -i "^access-control-allow-origin:" | tr -d '\r')
echo "    ${ACAO:-access-control-allow-origin: (header absent)}"

if echo "${ACAO}" | grep -qi "${EVIL_ORIGIN}"; then
  echo "    RESULT: [VULN] Attacker origin reflected — any site can make credentialed API requests."
else
  echo "    RESULT: [SAFE] Attacker origin NOT reflected."
fi
echo ""

# Test 2: Missing security headers
echo "[2] Checking for security headers..."
HEADERS_RESP=$(curl -si -X GET "${BASE}/projects" \
  -H "Origin: ${EVIL_ORIGIN}")

check_header() {
  local header="$1"
  local line
  line=$(echo "${HEADERS_RESP}" | grep -i "^${header}:" | tr -d '\r')
  if [ -n "${line}" ]; then
    echo "    [PRESENT] ${line}"
  else
    echo "    [MISSING] ${header}"
  fi
}

check_header "content-security-policy"
check_header "x-frame-options"
check_header "x-content-type-options"
check_header "strict-transport-security"
check_header "referrer-policy"
echo ""

echo "=== Summary ==="
echo "Re-run with VULN_MODE=false and ALLOWED_ORIGIN=http://localhost:5173 to verify hardening."
