#!/usr/bin/env bash
# security/scripts/01-xss-attack.sh
# Demonstrates the XSS vulnerability via the chat API.
# Run with VULN_MODE=true to see the attack succeed.
# Run with VULN_MODE=false to verify sanitization blocks it.

set -euo pipefail

BASE="${API_BASE:-http://localhost:8080}"
SESSION_ENDPOINT="${BASE}/api/chat/session"
MESSAGE_ENDPOINT="${BASE}/api/chat/message"

echo "=== XSS Attack Script ==="
echo "Target: ${BASE}"
echo ""

# Step 1: Create a session
echo "[1] Creating chat session..."
SESSION_RESP=$(curl -sf -X POST "${SESSION_ENDPOINT}" \
  -H "Content-Type: application/json") || {
  echo "ERROR: Could not reach ${SESSION_ENDPOINT} — is the server running?"
  exit 1
}
SESSION_ID=$(echo "${SESSION_RESP}" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)
echo "    session_id: ${SESSION_ID}"
echo ""

# Step 2: Inject XSS payload — use jq if available to guarantee valid JSON,
# otherwise fall back to a payload without embedded double quotes.
PAYLOAD='<img src=x onerror=alert(document.cookie)>'
echo "[2] Sending XSS payload:"
echo "    ${PAYLOAD}"
echo ""

if command -v jq &>/dev/null; then
  BODY=$(jq -n --arg sid "${SESSION_ID}" --arg msg "${PAYLOAD}" \
    '{"session_id": $sid, "message": $msg}')
else
  # Manual escape: replace " with \" inside the payload
  ESCAPED="${PAYLOAD//\"/\\\"}"
  BODY="{\"session_id\":\"${SESSION_ID}\",\"message\":\"${ESCAPED}\"}"
fi

RESPONSE=$(curl -sf -X POST "${MESSAGE_ENDPOINT}" \
  -H "Content-Type: application/json" \
  -d "${BODY}") || {
  echo "ERROR: POST to ${MESSAGE_ENDPOINT} failed — check server logs."
  exit 1
}

echo "[3] Raw response from server:"
echo "${RESPONSE}"
echo ""

# Step 3: Check if the onerror attribute survived in the response
if echo "${RESPONSE}" | grep -q "onerror"; then
  echo "RESULT: [VULN] onerror attribute present in response — XSS payload was NOT sanitized."
  echo "        Open the chat UI (VITE_VULN_MODE=true) and send the payload to see the alert fire."
else
  echo "RESULT: [SAFE] onerror attribute not found in response — payload was sanitized or removed."
fi
