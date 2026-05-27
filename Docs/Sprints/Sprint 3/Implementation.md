## Sprint 3 — Implementation Order

### Phase 1: Go HTTP Server (foundation for everything else)

**Step 1 — Create `web_server.go`**
Stand up a `net/http` server alongside the existing CLI. Register all routes, handle CORS, and validate the Cognito JWT from the `Authorization: Bearer <token>` header on protected routes. Start it from main.go (run in a goroutine next to `runagent()`). No third-party framework needed — stdlib `http.ServeMux` is sufficient.

**Step 2 — Build a simple project store**
Create a `store/` package. A `ProjectStore` struct holds an in-memory `map[string]*Project` guarded by a `sync.RWMutex`, and reads/writes each project as a JSON file under `data/projects/`. Load all JSON files on startup. This gives you persistence that survives restarts without pulling in a database.

```
Project {
  ID, Name, Description, Answers, Status (pending/generating/complete/failed),
  Messages []ChatMessage, TerraformScript string,
  Deployment *DeploymentResult
}
```

---

### Phase 2: Core Backend Endpoints

**Step 3 — `POST /projects`**
Accept `{ answers: {...} }`, create a `Project` record, save it, spawn a goroutine that runs the LLM generation loop, and immediately return `{ id: "..." }` to the caller. The frontend will receive real-time updates via WebSocket.

**Step 4 — Build the initial prompt from answers**
In the goroutine, format the answers dict into a structured system + user message (e.g., "You are an infrastructure engineer. The user's project has the following properties: name=X, type=Y, storage=Z..."). Use the existing `transport.Provider` (`anthropic.go` / `openai.go`) with a fresh `chat.Chat` instance per project.

**Step 5 — Generation loop with Terraform detection**
Run the `SendRequest` loop. After each assistant response, check if the content contains a valid Terraform configuration (simple heuristic: contains `resource "` or `terraform {` and valid HCL braces). If valid, extract it, save to `data/artifacts/<id>.tf`, mark project `status=complete`. If not, keep the loop running and wait for the next user message.

**Step 6 — `GET /projects`, `GET /projects/:id`, and `GET /projects/:id/ws`**
Return the project list via `GET /projects`. Keep `GET /projects/:id` as a plain HTTP endpoint. Add a `GET /projects/{id}/ws` WebSocket endpoint that upgrades the connection (requires a library — stdlib does not support WS upgrades; use `nhooyr.io/websocket` or `gorilla/websocket`) and pushes the full `Project` JSON blob to the connected client on every mutation: after each LLM response, after status changes, and after deployment completes. The JWT is validated from the `token` query parameter on the upgrade request (browsers cannot send custom headers during a WS handshake).

**Step 7 — `POST /projects/:id/messages`**
Accept `{ content: "..." }`, append the user message to the project's chat, trigger the next `SendRequest` call (in a goroutine), and return `200 OK`. The WebSocket will push the updated project state to the connected client once the response is ready.

**Step 8 — `GET /projects/:id/artifact`**
Read `data/artifacts/<id>.tf` from disk and return it as plain text.

**Step 9 — `POST /projects/:id/deploy` and `GET /projects/:id/deployment`**
On deploy: run `exec.Command("terraform", "plan")` in a temp dir with the artifact written to it. Capture stdout+stderr, record exit code, save as `DeploymentResult { Status, Output, Timestamp }` on the project. On GET: return that result.

---

### Phase 3: Frontend Wiring

**Step 10 — Update api.ts**
Add a `getAuthHeaders()` helper that calls Amplify's `fetchAuthSession()` and returns `{ Authorization: "Bearer <idToken>" }`. Update (or replace) all fetch calls to include these headers and point at the Go backend's base URL (configurable via `VITE_API_URL` env var, default `http://localhost:8080`).

**Step 11 — Wire `NewProjectPage.tsx` submit**
On "Create this project" click: call `POST /projects` with the answers object. Show a loading/disabled state on the button. On success, navigate to `/projects/:id/chat` with the returned project ID.

**Step 12 — Add `/projects/:id/chat` to `router.tsx`**
Add the route, passing the project ID as a param. Protect it under `AuthGuard`.

**Step 13 — Update `ChatPage.tsx`**
On mount, fetch `GET /projects/:id` and render existing messages. Set up a polling interval (`setInterval` every 3s) that re-fetches the project and appends any new messages. Replace the current session-based send with `POST /projects/:id/messages`. Show a "Script is ready" banner when `status === "complete"`.

**Step 14 — Deploy button and results**
In `ChatPage.tsx` (or a new `ProjectDetailPage`): show a "Deploy" button only when `status === "complete"` and `deployment` is null. On click, call `POST /projects/:id/deploy`, enter loading state, then poll `GET /projects/:id` to pick up the deployment result. Render result in a `<pre>` block with a green/red status badge.

**Step 15 — Wire `ProjectsPage.tsx`**
Replace the hardcoded empty array with a `GET /projects` call on mount.

---

### Sequence summary

```
1. web_server.go (HTTP server + JWT middleware)
2. store/ package (in-memory + JSON file persistence)
3. POST /projects (create + kick off generation goroutine)
4. Initial prompt construction + LLM loop
5. Terraform detection → save artifact
6. GET /projects, GET /projects/:id
7. POST /projects/:id/messages
8. GET /projects/:id/artifact
9. POST /projects/:id/deploy + GET /projects/:id/deployment
10. api.ts (JWT headers + base URL)
11. NewProjectPage submit → navigate to chat
12. router.tsx new route
13. ChatPage polling + message send
14. Deploy button + results display
15. ProjectsPage fetch
```

---

**Key decisions baked in:**
- **WebSockets over polling** — real-time push avoids the 3-second update delay; `ChatPage` connects to `GET /projects/{id}/ws` on mount and receives the full project state on every mutation. Requires adding a WebSocket library to the backend (stdlib does not support WS upgrades).
- **File-based persistence** — no database dependency, just JSON + `.tf` files under `data/`
- **Stdlib HTTP** — no Gin/Echo, keeps go.mod minimal
- **One goroutine per project** for LLM generation — simple, safe with the per-project mutex