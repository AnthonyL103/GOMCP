import { useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { confirmSignUp } from "aws-amplify/auth";

export default function VerifyPage() {
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const email = (location.state as { email?: string })?.email ?? "";

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await confirmSignUp({ username: email, confirmationCode: code });
      void navigate("/login");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Verification failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex h-screen items-center justify-center bg-bg px-4">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <span className="text-2xl font-semibold tracking-tight text-ink">
            GoMCP
          </span>
        </div>

        <div className="rounded-2xl border border-border bg-surface-raised px-8 py-10 shadow-sm">
          <h1 className="mb-2 text-xl font-semibold tracking-tight text-ink">
            Check your email
          </h1>
          <p className="mb-6 text-sm text-ink-secondary">
            We sent a 6-digit code to{" "}
            {email ? (
              <span className="font-medium text-ink">{email}</span>
            ) : (
              "your email address"
            )}
            .
          </p>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <label
                htmlFor="code"
                className="text-sm font-medium text-ink-secondary"
              >
                Verification code
              </label>
              <input
                id="code"
                type="text"
                inputMode="numeric"
                maxLength={6}
                required
                autoComplete="one-time-code"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                placeholder="000000"
                className="rounded-lg border border-border bg-surface px-4 py-2.5 text-center text-lg tracking-[0.5em] text-ink placeholder-ink-muted outline-none transition-colors focus:border-ink-muted focus:bg-surface-raised"
              />
            </div>

            {error && <p className="text-sm text-red-600">{error}</p>}

            <button
              type="submit"
              disabled={loading || code.length !== 6}
              className="mt-2 rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? "Verifying…" : "Confirm"}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
