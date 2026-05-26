import { useEffect, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import { LuSendHorizontal } from "react-icons/lu";
import {
  getProject,
  postMessage,
  deployProject,
  type Project,
} from "../api";

const POLL_INTERVAL_MS = 3000;

export default function ChatPage() {
  const { id } = useParams<{ id: string }>();

  const [project, setProject] = useState<Project | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Auto-scroll to bottom whenever messages change.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [project?.messages]);

  // Auto-resize textarea.
  useEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
  }, [input]);

  // Load project on mount, then poll every 3 s for updates.
  useEffect(() => {
    if (!id) return;

    async function fetchProject() {
      try {
        const p = await getProject(id!);
        setProject(p);
        setLoadError(null);

        // Stop polling once the project is in a terminal state with a
        // deployment result (nothing left to update).
        if (
          (p.status === "complete" || p.status === "failed") &&
          p.deployment
        ) {
          if (pollRef.current) {
            clearInterval(pollRef.current);
            pollRef.current = null;
          }
        }
      } catch (err: unknown) {
        setLoadError(err instanceof Error ? err.message : "Failed to load project");
      }
    }

    void fetchProject();
    pollRef.current = setInterval(() => void fetchProject(), POLL_INTERVAL_MS);

    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [id]);

  async function handleSend() {
    const trimmed = input.trim();
    if (!trimmed || sending || project?.status === "generating") return;
    setActionError(null);
    setInput("");
    setSending(true);
    try {
      await postMessage(id!, trimmed);
      // Optimistically add the user message; polling will add the response.
      setProject((prev) =>
        prev
          ? {
              ...prev,
              status: "generating",
              messages: [
                ...prev.messages,
                { role: "user", content: trimmed, created_at: new Date().toISOString() },
              ],
            }
          : prev,
      );
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : "Failed to send message");
    } finally {
      setSending(false);
    }
  }

  async function handleDeploy() {
    if (deploying) return;
    setActionError(null);
    setDeploying(true);
    try {
      await deployProject(id!);
      // Polling will pick up the deployment result.
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : "Deployment request failed");
    } finally {
      setDeploying(false);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void handleSend();
    }
  }

  const isGenerating = project?.status === "generating";
  const isComplete = project?.status === "complete";
  const inputDisabled = sending || isGenerating || !project;

  // ---- Loading / error state ----
  if (!project && !loadError) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="flex items-center gap-1.5">
          <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-ink-muted" />
          <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-ink-muted [animation-delay:150ms]" />
          <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-ink-muted [animation-delay:300ms]" />
        </div>
      </div>
    );
  }

  if (loadError) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <p className="text-sm text-red-500">{loadError}</p>
      </div>
    );
  }

  return (
    <>
      {/* Script-ready banner */}
      {isComplete && (
        <div className="flex items-center justify-between border-b border-border bg-surface-raised px-6 py-3">
          <span className="text-sm font-medium text-green-700">
            ✓ Terraform script is ready
          </span>
          {!project?.deployment && (
            <button
              onClick={() => void handleDeploy()}
              disabled={deploying}
              aria-label="Deploy Terraform script"
              className="rounded-lg bg-brand px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover disabled:opacity-60 disabled:cursor-not-allowed"
            >
              {deploying ? "Deploying…" : "Deploy"}
            </button>
          )}
        </div>
      )}

      {/* Message thread */}
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-3xl px-4 py-6 space-y-6">
          {project?.messages.map((msg, i) => (
            <div key={i}>
              {msg.role === "user" ? (
                <div className="flex justify-end">
                  <div className="max-w-[80%] rounded-2xl bg-bubble px-4 py-3 text-ink whitespace-pre-wrap">
                    {msg.content}
                  </div>
                </div>
              ) : (
                <div className="flex justify-start">
                  <div className="max-w-[80%] text-ink leading-relaxed whitespace-pre-wrap">
                    {msg.content}
                  </div>
                </div>
              )}
            </div>
          ))}

          {/* Generation in-progress indicator */}
          {isGenerating && (
            <div className="flex justify-start">
              <div className="flex items-center gap-1.5">
                <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-ink-muted" />
                <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-ink-muted [animation-delay:150ms]" />
                <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-ink-muted [animation-delay:300ms]" />
              </div>
            </div>
          )}

          {/* Deployment result */}
          {project?.deployment && (
            <div className="rounded-xl border border-border bg-surface-raised p-4 space-y-2">
              <div className="flex items-center gap-2">
                {project.deployment.status === "success" ? (
                  <span className="text-sm font-medium text-green-700">✓ Deployment succeeded</span>
                ) : (
                  <span className="text-sm font-medium text-red-600">✗ Deployment failed</span>
                )}
                <span className="text-xs text-ink-muted">
                  {new Date(project.deployment.timestamp).toLocaleString()}
                </span>
              </div>
              <pre className="overflow-x-auto rounded-lg bg-bg p-3 text-xs text-ink leading-relaxed whitespace-pre">
                {project.deployment.output}
              </pre>
            </div>
          )}

          <div ref={bottomRef} />
        </div>
      </div>

      {/* Input area */}
      <div className="px-4 pb-4">
        <div className="mx-auto max-w-3xl">
          {actionError && (
            <div className="mb-2 rounded-xl bg-red-50 px-4 py-2.5 text-sm text-red-600">
              {actionError}
            </div>
          )}
          <div className="rounded-2xl border border-border bg-surface-raised shadow-sm transition-shadow focus-within:shadow-md">
            <textarea
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={inputDisabled}
              placeholder={isGenerating ? "Generating…" : "Send a follow-up message"}
              rows={1}
              className="w-full resize-none bg-transparent px-5 pt-4 pb-3 text-base text-ink placeholder-ink-muted outline-none disabled:opacity-50"
            />
            <div className="flex items-center justify-end px-3 pb-3">
              <button
                onClick={() => void handleSend()}
                disabled={inputDisabled || !input.trim()}
                aria-label="Send message"
                className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand text-white transition-colors hover:bg-brand-hover disabled:opacity-30 disabled:cursor-not-allowed"
              >
                <LuSendHorizontal className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

