import { useEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { motion, AnimatePresence } from "framer-motion";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { vscDarkPlus } from "react-syntax-highlighter/dist/esm/styles/prism";

import type { Message, ToolCall, ToolResult } from "../api";

function ToolCallBlock({ toolCall }: { toolCall: ToolCall }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="mt-3 overflow-hidden rounded-xl border border-emerald-500/20 bg-emerald-500/5 text-xs font-mono">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-emerald-300 transition-colors hover:bg-emerald-500/10"
      >
        <span className="shrink-0 text-[11px]">⚙</span>
        <span className="font-semibold">{toolCall.tool_id}</span>
        <span className="ml-auto text-emerald-500">{toolCall.server_id}</span>
        <span className="h-3 w-3 text-[11px]">{open ? "▾" : "▸"}</span>
      </button>
      {open && (
        <div className="flex flex-col gap-2 border-t border-white/5 px-3 pb-3 pt-2">
          {toolCall.reasoning && (
            <p className="text-[11px] italic text-ink-muted">{toolCall.reasoning}</p>
          )}
          <pre className="whitespace-pre-wrap break-all text-[11px] text-ink-secondary">
            {JSON.stringify(toolCall.parameters, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

function ToolResultBlock({ toolResult }: { toolResult: ToolResult }) {
  const [open, setOpen] = useState(false);

  return (
    <div
      className={`mt-2 overflow-hidden rounded-xl border text-xs font-mono ${
        toolResult.is_error
          ? "border-red-500/20 bg-red-500/5"
          : "border-white/10 bg-white/5"
      }`}
    >
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-ink-secondary transition-colors hover:bg-white/5"
      >
        <span
          className={`h-2 w-2 shrink-0 rounded-full ${
            toolResult.is_error ? "bg-red-500" : "bg-emerald-500"
          }`}
        />
        <span>{toolResult.tool_id} result</span>
        <span className="ml-auto text-[11px]">{open ? "▾" : "▸"}</span>
      </button>
      {open && (
        <pre className="border-t border-white/5 px-3 pb-3 pt-2 whitespace-pre-wrap break-all text-[11px] text-ink-muted">
          {toolResult.content}
        </pre>
      )}
    </div>
  );
}

function MarkdownContent({ content }: { content: string }) {
  return (
    <div className="prose prose-invert prose-sm max-w-none prose-p:my-2 prose-pre:!bg-transparent prose-pre:!p-0">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          code({ inline, className, children, ...props }: any) {
            const match = /language-(\w+)/.exec(className || "");

            if (!inline && match) {
              return (
                <SyntaxHighlighter
                  {...props}
                  style={vscDarkPlus}
                  language={match[1]}
                  PreTag="div"
                  className="my-3 rounded-xl border border-white/10 !bg-black !p-3 text-[12px] font-mono shadow-inner"
                >
                  {String(children).replace(/\n$/, "")}
                </SyntaxHighlighter>
              );
            }

            return (
              <code
                {...props}
                className={`${className ?? ""} rounded bg-white/10 px-1.5 py-0.5 font-mono text-[0.85em] text-emerald-300`}
              >
                {children}
              </code>
            );
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}

function MessageBubble({ message }: { message: Message }) {
  const isUser = message.role === "user";

  return (
    <motion.div
      initial={{ opacity: 0, y: 10, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.22 }}
      className={`flex gap-3 ${isUser ? "flex-row-reverse" : ""}`}
    >
      <div
        className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full border ${
          isUser
            ? "border-blue-500/20 bg-blue-500/10 text-blue-400"
            : "border-emerald-500/20 bg-emerald-500/10 text-emerald-400"
        }`}
      >
        {isUser ? "U" : "AG"}
      </div>

      <div className={`flex w-full max-w-[85%] flex-col gap-1 ${isUser ? "items-end" : ""}`}>
        <span className="text-xs font-medium text-ink-muted">
          {isUser ? "You" : "Agent"}
        </span>

        <div
          className={`rounded-2xl border px-4 py-3 text-sm leading-relaxed shadow-sm ${
            isUser
              ? "rounded-tr-sm border-blue-500 bg-blue-600 text-white"
              : "rounded-tl-sm border-border bg-surface-raised text-ink"
          }`}
        >
          {!isUser && message.tool_call && <ToolCallBlock toolCall={message.tool_call} />}
          {!isUser && message.tool_result && <ToolResultBlock toolResult={message.tool_result} />}
          {!isUser && message.content && <MarkdownContent content={message.content} />}
          {isUser && <div className="whitespace-pre-wrap">{message.content}</div>}
        </div>
      </div>
    </motion.div>
  );
}

export default function ChatPanel({
  messages,
  isBootstrapping,
  isSending,
  onSend,
  title,
  subtitle,
}: {
  messages: Message[];
  isBootstrapping: boolean;
  isSending: boolean;
  onSend: (message: string) => void;
  title: string;
  subtitle?: string;
}) {
  const [input, setInput] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages, isBootstrapping, isSending]);

  const canSend = useMemo(() => !isBootstrapping && !isSending && input.trim().length > 0, [input, isBootstrapping, isSending]);

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    if (!canSend) return;
    const trimmed = input.trim();
    setInput("");
    onSend(trimmed);
  }

  return (
    <div className="flex h-full flex-col">
      <div className="sticky top-0 z-10 border-b border-border bg-surface-raised/80 px-5 py-4 backdrop-blur-md">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl border border-emerald-500/20 bg-emerald-500/10 text-emerald-400">
            AG
          </div>
          <div>
            <h1 className="text-base font-semibold tracking-tight text-ink">{title}</h1>
            <p className="text-sm text-ink-secondary">{subtitle ?? "Chat with the agent and watch tool calls appear live."}</p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-6">
        <div className="mx-auto flex max-w-4xl flex-col gap-5">
          <AnimatePresence initial={false}>
            {messages.map((message, index) => (
              <MessageBubble key={`${message.timestamp}-${index}`} message={message} />
            ))}

            {isBootstrapping && (
              <motion.div
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                className="flex items-center gap-3 rounded-2xl border border-border bg-surface-raised px-4 py-4 text-sm text-ink-secondary"
              >
                <span className="inline-block h-3 w-3 animate-pulse rounded-full bg-emerald-400" />
                Agent is thinking about your project request…
              </motion.div>
            )}
          </AnimatePresence>

          <div ref={bottomRef} />
        </div>
      </div>

      <div className="border-t border-border bg-surface-raised/80 px-4 py-4 backdrop-blur-sm">
        <form onSubmit={handleSubmit} className="mx-auto flex max-w-4xl items-end gap-3">
          <div className="flex-1 rounded-2xl border border-border bg-bg/90 shadow-sm transition-shadow focus-within:shadow-md">
            <textarea
              value={input}
              onChange={(event) => setInput(event.target.value)}
              placeholder={isBootstrapping ? "Waiting for the agent…" : "Ask a follow-up or refine the request"}
              disabled={isBootstrapping || isSending}
              rows={1}
              className="max-h-48 min-h-14 w-full resize-none bg-transparent px-4 py-4 text-sm text-ink outline-none placeholder:text-ink-muted disabled:opacity-60"
            />
          </div>
          <button
            type="submit"
            disabled={!canSend}
            className="flex h-14 min-w-14 items-center justify-center rounded-2xl bg-brand px-5 text-white transition-colors hover:bg-brand-hover disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSending ? (
              <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white" />
            ) : (
              <span aria-hidden="true">➤</span>
            )}
          </button>
        </form>
      </div>
    </div>
  );
}