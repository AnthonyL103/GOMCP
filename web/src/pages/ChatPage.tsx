import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import ChatPanel from "../components/ChatPanel";
import { connectToolStream, createChatSessionId, getCurrentUserId, sendChatPrompt, type Message } from "../api";

type ChatLocationState = {
  sessionId?: string;
  initialPrompt?: string;
  projectName?: string;
};

const EMPTY_MESSAGE = (role: "user" | "assistant", content: string): Message => ({
  role,
  content,
  timestamp: new Date().toISOString(),
});

export default function ChatPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const state = (location.state ?? {}) as ChatLocationState;

  const sessionId = useMemo(() => state.sessionId ?? id ?? createChatSessionId(), [state.sessionId, id]);
  const storageKey = `gomcp-chat:${sessionId}`;

  const [messages, setMessages] = useState<Message[]>([]);
  const [isBootstrapping, setIsBootstrapping] = useState(true);
  const [isSending, setIsSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [userId, setUserId] = useState<string | null | undefined>(undefined);
  const didStartRef = useRef(false);

  const initialPrompt = state.initialPrompt ?? sessionStorage.getItem(storageKey) ?? "";

  useEffect(() => {
    const disconnect = connectToolStream((message) => {
      setMessages((current) => [...current, message]);
    });

    return disconnect;
  }, []);

  useEffect(() => {
    void (async () => {
      const currentUserId = await getCurrentUserId();
      setUserId(currentUserId);
    })();
  }, []);

  useEffect(() => {
    if (userId === undefined) return;
    if (!initialPrompt || didStartRef.current) return;
    didStartRef.current = true;
    sessionStorage.setItem(storageKey, initialPrompt);

    setMessages([EMPTY_MESSAGE("user", initialPrompt)]);
    setIsBootstrapping(true);
    setError(null);

    void (async () => {
      try {
        const reply = await sendChatPrompt(initialPrompt, { sessionId, userId: userId ?? undefined });
        setMessages((current) => [...current, reply]);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Failed to start chat");
      } finally {
        setIsBootstrapping(false);
      }
    })();
  }, [initialPrompt, sessionId, storageKey, userId]);

  async function handleSend(message: string) {
    if (isSending) return;
    setError(null);
    setIsSending(true);
    setMessages((current) => [...current, EMPTY_MESSAGE("user", message)]);

    try {
      const reply = await sendChatPrompt(message, { sessionId, userId: userId ?? undefined });
      setMessages((current) => [...current, reply]);
    } catch (err: unknown) {
      setMessages((current) => [
        ...current,
        EMPTY_MESSAGE("assistant", err instanceof Error ? err.message : "Failed to send message"),
      ]);
      setError(err instanceof Error ? err.message : "Failed to send message");
    } finally {
      setIsSending(false);
      setIsBootstrapping(false);
    }
  }

  function handleReset() {
    sessionStorage.removeItem(storageKey);
    void navigate("/projects/new");
  }

  return (
    <div className="relative flex flex-1 flex-col overflow-hidden">
      {error && (
        <div className="border-b border-red-500/20 bg-red-500/5 px-4 py-2 text-sm text-red-500">
          {error}
        </div>
      )}

      <div className="flex-1 overflow-hidden">
        <ChatPanel
          title={state.projectName ? `${state.projectName} chat` : `Chat session ${sessionId.slice(0, 8)}`}
          subtitle="The agent responds in markdown and tool output arrives live over the websocket."
          messages={messages}
          isBootstrapping={isBootstrapping}
          isSending={isSending}
          onSend={handleSend}
        />
      </div>

      {!initialPrompt && messages.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center bg-bg/80 backdrop-blur-sm">
          <div className="rounded-2xl border border-border bg-surface-raised px-6 py-5 text-center shadow-lg">
            <h2 className="text-base font-semibold text-ink">No project prompt found</h2>
            <p className="mt-1 text-sm text-ink-secondary">
              Start a new project to generate the initial chat prompt.
            </p>
            <button
              type="button"
              onClick={handleReset}
              className="mt-4 rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-hover"
            >
              Create a project
            </button>
          </div>
        </div>
      )}
    </div>
  );
}