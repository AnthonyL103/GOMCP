# Sprint 3 — End-to-End Workflow

---

**Wire up the frontend to the backend**
The frontend and backend are currently running independently. Add a base API client in `web/src/api.ts` that points at the Go backend. Authenticated requests should attach the Cognito JWT from the active session as a Bearer token in the `Authorization` header.

Acceptance criteria: Open the browser network tab → submit the new project form → a POST request fires to the Go backend with the correct `Authorization` header and a non-empty JSON body. A missing or expired session should result in a 401 being surfaced, not a silent failure.

---

**Send answers object on project creation**
When the user clicks "Create this project" on the review screen, `NewProjectPage.tsx` should POST the collected `answers` object to `POST /projects`. The button should show a loading state while the request is in flight and be disabled to prevent double-submits.

Acceptance criteria: Complete the new project wizard → click "Create this project" → the network tab shows a single POST to `/projects` with the full answers object as JSON. Clicking the button a second time before the response returns has no effect.

---

**Auto-inject form answers into the initial prompt**
The backend receives the answers object from the frontend and constructs a structured prompt that embeds those answers before sending anything to the LLM. The answers should be formatted clearly so the model has full context without requiring the user to re-explain their project.

Acceptance criteria: Submit a project with known answers → inspect backend logs or the LLM request payload → the raw prompt contains each answer value embedded in the correct position. Submitting a project with different answers produces a materially different prompt.

---

**Backend responds to the initial prompt**
After injecting the answers, the backend sends the first message to the LLM and streams or returns the model's initial response back to the client. This is the starting message of the generation conversation and should be stored as the first message in the project's chat history.

Acceptance criteria: Submit a project → the frontend receives a non-empty response from the backend. The response is the model's reply to the injected prompt, not a static placeholder. The response is saved and retrievable for the session.

---

**Interactive generation chat loop**
The generation process is interactive: after the initial response, the user can send follow-up messages and the backend will continue the conversation, maintaining context from all prior turns. The loop continues until the user or the model signals completion.

Acceptance criteria: Submit a project → receive the first response → send a follow-up message → receive a reply that references context from the earlier turn. The backend does not reset the conversation state between messages.

---

**Continue until a valid deployable Terraform script is produced**
The backend evaluates each model response to determine whether it contains a complete, syntactically valid Terraform configuration. If not, the generation loop continues. Once a valid script is detected, the session is flagged as complete and generation stops.

Acceptance criteria: Walk through the generation loop to completion → verify the backend marks the session done only when the output contains a parseable Terraform configuration. A response that contains partial or malformed HCL does not trigger completion.

---

**Frontend chat view for the generation conversation**
After project creation kicks off, navigate the user to a dedicated chat page for that project (e.g. `/projects/:id/chat`). This view displays the message thread between the user and the model as generation progresses, with an input field for follow-up messages.

Acceptance criteria: Submit a project → land on the chat page for that project → see the model's first response rendered in the thread. Send a follow-up message → see it appear in the thread followed by the model's reply. Refreshing the page re-renders the existing message history.

---

**Persist the generated Terraform script**
Once the backend detects a valid Terraform configuration in the model output, extract it and save it as an artifact associated with the project (e.g. stored on disk or in a database record). The script must be retrievable later by project ID.

Acceptance criteria: Complete a generation session → query the backend for the project's artifact (e.g. `GET /projects/:id/artifact`) → receive the Terraform script as the response body. The script content matches what the model produced. Re-fetching after a server restart still returns the same script.

---

**Notify the user when the script is ready**
When the Terraform script has been persisted, the frontend should surface a clear indication that the script is available — for example, a banner, a status badge on the project, or an in-chat system message. The user should not have to poll or guess when generation is complete.

Acceptance criteria: Complete the generation loop → without manually refreshing, a visible indicator appears in the UI confirming the script is ready. The indicator links to or reveals the script content.

---

**Initiate a deployment from the frontend**
Add a "Deploy" button to the project chat or detail view, visible only once a script artifact exists. Clicking it sends a request to the backend to begin a Terraform deployment for this project. The button should transition to a loading/in-progress state immediately after being clicked.

Acceptance criteria: Open a project with a completed script → see the Deploy button → click it → the button enters a loading state and a deployment request is received by the backend. The button is not visible on projects that do not yet have a completed script.

---

**Backend executes the deployment via `terraform plan`**
When the backend receives a deploy request, it runs `terraform plan` (and optionally `terraform apply`) against the saved script in a sandboxed working directory. stdout and stderr are captured in their entirety for surfacing to the user.

Acceptance criteria: Trigger a deployment from the frontend → confirm in backend logs that `terraform plan` was executed with the correct working directory. Both a successful plan and a plan with errors produce captured output that is stored and associated with the deployment record.

---

**Surface deployment output to the user**
The full stdout/stderr output from the Terraform run is sent back to the frontend and displayed in the project view — rendered in a fixed-width block so formatting is preserved. The output should be available as soon as the run completes, pushed to the frontend via the project's WebSocket connection.

Acceptance criteria: Trigger a deployment → wait for it to finish → see the raw Terraform output displayed in the UI with whitespace and line breaks intact. Output from a failing plan and a passing plan are both displayed correctly.

---

**Backend distinguishes deployment success from failure**
After the Terraform run completes, the backend checks the exit code and marks the deployment record as either `success` or `failure`. This status is returned to the frontend alongside the output and must be stored persistently.

Acceptance criteria: Trigger a deployment that succeeds → deployment record shows `success`. Introduce a deliberate error in the script → trigger a deployment → record shows `failure`. Both statuses survive a server restart (i.e. are persisted, not just in-memory).

---

**Frontend deployment results view**
The project view shows the outcome of the most recent deployment: the status (`success` or `failure`) displayed with a clear visual treatment (e.g. green/red badge), the full Terraform output, and the timestamp of the run. If no deployment has been attempted, this section is hidden.

Acceptance criteria: Complete a successful deployment → see a green success indicator and the Terraform output in the UI. Trigger a failed deployment → see a red failure indicator and the error output. A project with no deployments shows no deployment results section.
