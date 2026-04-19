export async function createSession(): Promise<string> {
  const res = await fetch("/api/chat/session", { method: "POST" });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(body || `Failed to create session (${res.status})`);
  }
  const data: { session_id: string } = await res.json();
  return data.session_id;
}

export async function sendMessage(
  sessionId: string,
  message: string,
): Promise<string> {
  const res = await fetch("/api/chat/message", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ session_id: sessionId, message }),
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(body || `Request failed (${res.status})`);
  }
  const data: { response: string } = await res.json();
  return data.response;
}
