import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { LuSendHorizontal } from "react-icons/lu";

function getGreeting(): string {
  const h = new Date().getHours();
  if (h < 12) return "Good morning";
  if (h < 17) return "Good afternoon";
  return "Good evening";
}

export default function LandingPage() {
  const [input, setInput] = useState("");
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
  }, [input]);

  function handleSend() {
    const trimmed = input.trim();
    if (!trimmed) return;
    void navigate("/chat", { state: { firstMessage: trimmed } });
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  return (
    <div className="flex flex-1 flex-col items-center justify-center px-4">
      <div className="mb-8 text-center">
        <h1 className="text-4xl font-semibold tracking-tight text-stone-800">
          {getGreeting()}.
        </h1>
      </div>
      <div className="w-full max-w-3xl">
        <div className="rounded-2xl border border-stone-200 bg-white shadow-sm transition-shadow focus-within:shadow-md">
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="What are we building?"
            rows={1}
            className="w-full resize-none bg-transparent px-5 pt-4 pb-3 text-base text-stone-800 placeholder-stone-400 outline-none"
          />
          <div className="flex items-center justify-end px-3 pb-3">
            <button
              onClick={handleSend}
              disabled={!input.trim()}
              className="flex h-8 w-8 items-center justify-center rounded-lg bg-stone-800 text-white transition-colors hover:bg-stone-700 disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <LuSendHorizontal className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
