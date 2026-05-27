import { fetchAuthSession, getCurrentUser } from "aws-amplify/auth";

// Base URL comes from the Vite env var; falls back to localhost for local dev.
export const API_URL =
  (import.meta.env.VITE_API_URL as string | undefined) ?? "http://localhost:8080";

export const WS_URL = API_URL.replace(/^http/, "ws");

// -------------------------------------------------------------------
// Go backend types (mirrors chat package)
// -------------------------------------------------------------------

export interface ToolCall {
  server_id: string;
  tool_id: string;
  handler: string;
  parameters: Record<string, unknown>;
  reasoning: string;
  tool_use_id: string;
}

export interface ToolResult {
  server_id: string;
  tool_id: string;
  content: string;
  is_error: boolean;
  tool_use_id: string;
}

export interface Message {
  role: "user" | "assistant";
  content: string;
  timestamp: string;
  tool_call?: ToolCall;
  tool_result?: ToolResult;
}

export interface DeploymentResult {
  status: string;
  output: string;
  timestamp: string;
}

export interface Project {
  id: string;
  user_id?: string;
  name: string;
  answers: Record<string, unknown>;
  status: string;
  messages: Message[];
  terraform_script?: string;
  deployment?: DeploymentResult;
  created_at: string;
  updated_at: string;
}

export interface ChatRequestContext {
  sessionId: string;
  userId?: string;
}

// -------------------------------------------------------------------
// Project prompt helpers
// -------------------------------------------------------------------

export interface ProjectAnswers {
  name: string;
  description: string;
  projectType: string;
  dataStorage: string;
  background: string;
  audience: string;
  usage: string;
  reliability: string;
  sensitiveData: string;
  location: string;
  domain: string;
}

const ANSWER_LABELS: Record<keyof ProjectAnswers, string> = {
  name: "Project name",
  description: "Description",
  projectType: "Project type",
  dataStorage: "Data storage",
  background: "Background tasks",
  audience: "Audience",
  usage: "Expected usage",
  reliability: "Reliability needs",
  sensitiveData: "Sensitive data",
  location: "User location",
  domain: "Domain",
};

export function buildProjectPrompt(answers: ProjectAnswers): string {
  const lines = Object.entries(answers)
    .filter(([, value]) => value.trim() !== "")
    .map(([key, value]) => `- ${ANSWER_LABELS[key as keyof ProjectAnswers]}: ${value}`);

  return [
    "Here is the user project request:",
    "",
    ...lines,
    "",
    "Please help the user design and implement this project. Start by explaining the plan and ask follow-up questions if needed. Return markdown when useful, and include terraform code blocks when you are ready to propose infrastructure.",
  ].join("\n");
}

export function createChatSessionId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `chat-${Date.now()}`;
}

export async function getCurrentUserId(): Promise<string | null> {
  try {
    const user = await getCurrentUser();
    return user.username?.trim() || null;
  } catch {
    return null;
  }
}

async function getAuthToken(): Promise<string | null> {
  try {
    const session = await fetchAuthSession();
    return session.tokens?.idToken?.toString() ?? session.tokens?.accessToken?.toString() ?? null;
  } catch {
    return null;
  }
}

async function authorizedHeaders(): Promise<Record<string, string>> {
  const token = await getAuthToken();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

function buildChatRequestBody(message: string, context?: ChatRequestContext): Record<string, unknown> {
  const body: Record<string, unknown> = { message };
  if (context?.sessionId) body.session_id = context.sessionId;
  if (context?.userId) body.user_id = context.userId;
  return body;
}

// -------------------------------------------------------------------
// Chat REST + WS
// -------------------------------------------------------------------

export async function sendChatPrompt(message: string, context?: ChatRequestContext): Promise<Message> {
  const res = await fetch(`${API_URL}/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(buildChatRequestBody(message, context)),
  });

  

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Request failed (${res.status})`);
  }

  return res.json() as Promise<Message>;
}

export async function listProjects(): Promise<Project[]> {
  const headers = await authorizedHeaders();
  const res = await fetch(`${API_URL}/projects`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Request failed (${res.status})`);
  }

  return res.json() as Promise<Project[]>;
}

export function connectToolStream(onMessage: (msg: Message) => void): () => void {
  const ws = new WebSocket(`${WS_URL}/ws`);

  ws.onmessage = (event) => {
    try {
      onMessage(JSON.parse(event.data as string) as Message);
      console.log("Received message from tool stream:", event.data);
    } catch {
      // ignore malformed frames
    }
  };

  return () => ws.close();
}
