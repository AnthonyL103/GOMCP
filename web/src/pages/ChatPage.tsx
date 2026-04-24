import { useEffect, useRef, useState } from "react";
import { useLocation } from "react-router-dom";
import { LuSendHorizontal } from "react-icons/lu";
import { createSession, sendMessage } from "../api";

interface Message {
  role: "user" | "assistant";
  content: string;
}

export default function ChatPage() {
  const { state } = useLocation();
  const firstMessage = (state as { firstMessage?: string } | null)?.firstMessage;

  const [sessionId, setSessionId] = useState<string | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const initialized = useRef(false);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  useEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
  }, [input]);

  // init session on mount, then send first message if present
  useEffect(() => {
    if (initialized.current) return;
    initialized.current = true;

    const initialMsg = firstMessage;

    // Show the user's message immediately, before any async work
    if (initialMsg) {
      setMessages([{ role: "user", content: initialMsg }]);
      setLoading(true);
    }

    createSession()
      .then(async (id) => {
        setSessionId(id);
        if (!initialMsg) return;
        try {
          const response = await sendMessage(id, initialMsg);
          setMessages((prev) => [
            ...prev,
            { role: "assistant", content: response },
          ]);
        } catch (err: unknown) {
          setError(err instanceof Error ? err.message : "Something went wrong");
        } finally {
          setLoading(false);
        }
      })
      .catch(() => {
        setSessionId("offline");
        setLoading(false);
      });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleSend() {
    const trimmed = input.trim();
    if (!trimmed || loading || !sessionId) return;
    setError(null);
    setInput("");
    setLoading(true);
    setMessages((prev) => [...prev, { role: "user", content: trimmed }]);
    try {
      const response = await sendMessage(sessionId, trimmed);
      setMessages((prev) => [
        ...prev,
        { role: "assistant", content: response },
      ]);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setLoading(false);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void handleSend();
    }
  }

  return (
    <>
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-3xl px-4 py-6 space-y-6">
          {messages.map((msg, i) => (
            <div key={i}>
              {msg.role === "user" ? (
                <div className="flex justify-end">
                  <div className="max-w-[80%] rounded-2xl bg-stone-200/70 px-4 py-3 text-stone-900 whitespace-pre-wrap">
                    {msg.content}
                  </div>
                </div>
              ) : (
                <div className="flex justify-start">
                  <div className="max-w-[80%] text-stone-800 leading-relaxed whitespace-pre-wrap">
                    {msg.content}
                  </div>
                </div>
              )}
            </div>
          ))}

          {loading && (
            <div className="flex justify-start">
              <div className="flex items-center gap-1.5">
                <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-stone-400" />
                <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-stone-400 [animation-delay:150ms]" />
                <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-stone-400 [animation-delay:300ms]" />
              </div>
            </div>
          )}

          <div ref={bottomRef} />
        </div>
      </div>

      <div className="px-4 pb-4">
        <div className="mx-auto max-w-3xl">
          {error && (
            <div className="mb-2 rounded-xl bg-red-50 px-4 py-2.5 text-sm text-red-600">
              {error}
            </div>
          )}
          <div className="rounded-2xl border border-stone-200 bg-white shadow-sm transition-shadow focus-within:shadow-md">
            <textarea
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={loading || !sessionId}
              placeholder="What are we building?"
              rows={1}
              className="w-full resize-none bg-transparent px-5 pt-4 pb-3 text-base text-stone-800 placeholder-stone-400 outline-none disabled:opacity-50"
            />
            <div className="flex items-center justify-end px-3 pb-3">
              <button
                onClick={() => void handleSend()}
                disabled={loading || !sessionId || !input.trim()}
                className="flex h-8 w-8 items-center justify-center rounded-lg bg-stone-800 text-white transition-colors hover:bg-stone-700 disabled:opacity-30 disabled:cursor-not-allowed"
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
