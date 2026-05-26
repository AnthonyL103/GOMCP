import { fetchAuthSession } from "aws-amplify/auth";

// Base URL comes from the Vite env var; falls back to localhost for local dev.
const API_BASE =
  (import.meta.env.VITE_API_URL as string | undefined) ?? "http://localhost:8080";

// ---------------------------------------------------------------------------
// Types — mirror the Go store.Project struct (snake_case JSON tags)
// ---------------------------------------------------------------------------

export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
  created_at: string;
}

export interface DeploymentResult {
  status: "success" | "failure";
  output: string;
  timestamp: string;
}

export interface Project {
  id: string;
  name: string;
  answers: Record<string, string>;
  status: "pending" | "generating" | "complete" | "failed";
  messages: ChatMessage[];
  terraform_script?: string;
  deployment?: DeploymentResult;
  created_at: string;
  updated_at: string;
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

async function authHeaders(): Promise<Record<string, string>> {
  const session = await fetchAuthSession();
  const token = session.tokens?.idToken?.toString();
  if (!token) throw new Error("Not authenticated");
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
  };
}

async function apiRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = await authHeaders();
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: { ...headers, ...(init.headers as Record<string, string> | undefined) },
  });

  if (res.status === 401) {
    // Token missing or expired — send the user back to login.
    window.location.href = "/login";
    throw new Error("Session expired. Redirecting to login…");
  }

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    let message = text;
    try {
      message = (JSON.parse(text) as { error?: string }).error ?? text;
    } catch {
      // keep raw text as the message
    }
    throw new Error(message || `Request failed (${res.status})`);
  }

  return res.json() as Promise<T>;
}

// Variant for endpoints that return plain text (e.g. the Terraform artifact).
async function apiRequestText(path: string): Promise<string> {
  const headers = await authHeaders();
  const res = await fetch(`${API_BASE}${path}`, { headers });
  if (res.status === 401) {
    window.location.href = "/login";
    throw new Error("Session expired");
  }
  if (!res.ok) throw new Error(`Request failed (${res.status})`);
  return res.text();
}

// ---------------------------------------------------------------------------
// Public API functions
// ---------------------------------------------------------------------------

export function createProject(
  answers: Record<string, string>,
): Promise<{ id: string }> {
  return apiRequest("/projects", {
    method: "POST",
    body: JSON.stringify({ answers }),
  });
}

export function listProjects(): Promise<Project[]> {
  return apiRequest<Project[]>("/projects");
}

export function getProject(id: string): Promise<Project> {
  return apiRequest<Project>(`/projects/${id}`);
}

export function postMessage(projectId: string, content: string): Promise<void> {
  return apiRequest<void>(`/projects/${projectId}/messages`, {
    method: "POST",
    body: JSON.stringify({ content }),
  });
}

export function getArtifact(projectId: string): Promise<string> {
  return apiRequestText(`/projects/${projectId}/artifact`);
}

export function deployProject(projectId: string): Promise<void> {
  return apiRequest<void>(`/projects/${projectId}/deploy`, { method: "POST" });
}
