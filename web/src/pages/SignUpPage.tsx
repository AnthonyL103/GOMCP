import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { signUp } from "aws-amplify/auth";
import { IoArrowBack } from "react-icons/io5";
import { LuEye, LuEyeClosed } from "react-icons/lu";
import { FiCheckCircle } from "react-icons/fi";

export default function SignUpPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const navigate = useNavigate();

  const passwordChecks = [
    { label: "At least 8 characters", met: password.length >= 8 },
    { label: "Uppercase letter", met: /[A-Z]/.test(password) },
    { label: "Lowercase letter", met: /[a-z]/.test(password) },
    { label: "Number", met: /[0-9]/.test(password) },
    { label: "Special character", met: /[^A-Za-z0-9]/.test(password) },
  ];

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (password !== confirm) {
      setError("Passwords do not match.");
      return;
    }
    setLoading(true);
    try {
      await signUp({
        username: email,
        password,
        options: { userAttributes: { email } },
      });
      void navigate("/verify", { state: { email } });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Sign up failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="relative flex h-screen items-center justify-center bg-bg px-4">
      <Link
        to="/"
        className="absolute left-4 top-4 flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm text-ink-muted transition-colors hover:bg-border-subtle hover:text-ink-secondary"
      >
        <IoArrowBack className="h-4 w-4" />
        Home
      </Link>
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <span className="text-2xl font-semibold tracking-tight text-ink">
            GoMCP
          </span>
        </div>

        <div className="rounded-2xl border border-border bg-surface-raised px-8 py-10 shadow-sm">
          <h1 className="mb-6 text-xl font-semibold tracking-tight text-ink">
            Create your account
          </h1>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <label
                htmlFor="email"
                className="text-sm font-medium text-ink-secondary"
              >
                Email
              </label>
              <input
                id="email"
                type="email"
                required
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                className="rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-ink placeholder-ink-muted outline-none transition-colors focus:border-ink-muted focus:bg-surface-raised"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label
                htmlFor="password"
                className="text-sm font-medium text-ink-secondary"
              >
                Password
              </label>
              <div className="relative">
                <input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  required
                  autoComplete="new-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="w-full rounded-lg border border-border bg-surface px-4 py-2.5 pr-10 text-sm text-ink placeholder-ink-muted outline-none transition-colors focus:border-ink-muted focus:bg-surface-raised"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  tabIndex={-1}
                  aria-label={showPassword ? "Hide password" : "Show password"}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-ink-muted transition-colors hover:text-ink-secondary"
                >
                  {showPassword ? (
                    <LuEye className="h-4 w-4" />
                  ) : (
                    <LuEyeClosed className="h-4 w-4" />
                  )}
                </button>
              </div>
              {/* Password requirements */}
                <ul className="mt-1.5 flex flex-col gap-1">
                  {passwordChecks.map((c) => (
                    <li key={c.label} className="flex items-center gap-2">
                      <FiCheckCircle
                        className={`h-3.5 w-3.5 shrink-0 transition-colors ${
                          c.met ? "text-green-500" : "text-ink-muted"
                        }`}
                      />
                      <span
                        className={`text-xs transition-colors ${
                          c.met ? "text-ink-secondary" : "text-ink-muted"
                        }`}
                      >
                        {c.label}
                      </span>
                    </li>
                  ))}
                </ul>
            </div>

            <div className="flex flex-col gap-1.5">
              <label
                htmlFor="confirm"
                className="text-sm font-medium text-ink-secondary"
              >
                Confirm password
              </label>
              <div className="relative">
                <input
                  id="confirm"
                  type={showConfirm ? "text" : "password"}
                  required
                  autoComplete="new-password"
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  placeholder="••••••••"
                  className="w-full rounded-lg border border-border bg-surface px-4 py-2.5 pr-10 text-sm text-ink placeholder-ink-muted outline-none transition-colors focus:border-ink-muted focus:bg-surface-raised"
                />
                <button
                  type="button"
                  onClick={() => setShowConfirm((v) => !v)}
                  tabIndex={-1}
                  aria-label={showConfirm ? "Hide password" : "Show password"}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-ink-muted transition-colors hover:text-ink-secondary"
                >
                  {showConfirm ? (
                    <LuEye className="h-4 w-4" />
                  ) : (
                    <LuEyeClosed className="h-4 w-4" />
                  )}
                </button>
              </div>
            </div>

            {error && <p className="text-sm text-red-600">{error}</p>}

            <button
              type="submit"
              disabled={loading}
              className="mt-2 rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? "Creating account…" : "Create account"}
            </button>
          </form>

          <p className="mt-6 text-center text-sm text-ink-secondary">
            Already have an account?{" "}
            <Link
              to="/login"
              className="font-medium text-ink hover:underline"
            >
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
