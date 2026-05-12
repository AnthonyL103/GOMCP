import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { getCurrentUser } from "aws-amplify/auth";
import Orb from '../components/Orb';

const FEATURES = [
  {
    title: "Describe your project",
    description:
      "Answer a handful of plain-English questions about what you're building, who it's for, and how reliable it needs to be.",
  },
  {
    title: "We generate the architecture",
    description:
      "GoMCP translates your answers into a production-ready infrastructure plan — databases, compute, networking, and more.",
  },
  {
    title: "Deploy with one click",
    description:
      "Terraform files are generated and validated automatically. Connect your AWS account and ship whenever you're ready.",
  },
];

export default function LandingPage() {
  const [isAuthed, setIsAuthed] = useState(false);

  useEffect(() => {
    getCurrentUser()
      .then(() => setIsAuthed(true))
      .catch(() => {});
  }, []);

  return (
    <div className="min-h-screen bg-bg text-ink">
      {/* Nav */}
      <header className="absolute inset-x-0 top-0 z-20 border-b border-border/40 backdrop-blur-md">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-6">
          <span className="text-base font-semibold tracking-tight text-ink">
            GoMCP
          </span>
          <nav className="flex items-center gap-2">
            {isAuthed ? (
              <Link
                to="/projects"
                className="rounded-lg bg-brand px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover"
              >
                My Projects
              </Link>
            ) : (
              <>
                <Link
                  to="/login"
                  className="rounded-lg px-4 py-1.5 text-sm font-medium text-ink-secondary transition-colors hover:bg-border-subtle hover:text-ink"
                >
                  Sign in
                </Link>
                <Link
                  to="/signup"
                  className="rounded-lg bg-brand px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover"
                >
                  Get started
                </Link>
              </>
            )}
          </nav>
        </div>
      </header>

      {/* Hero — Orb fills the background */}
      <section className="relative overflow-hidden">
        {/* Orb background */}
        <div className="absolute inset-0 mt-10">
          <Orb
            hoverIntensity={2}
            rotateOnHover
            hue={0}
            forceHoverState={false}
            backgroundColor="#050714"
          />
        </div>
        {/* Fade the orb into the page bg at the bottom */}
        <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-40 bg-gradient-to-t from-bg to-transparent" />
        {/* Text overlay — pointer-events-none so mouse events reach the Orb behind it */}
        <div className="pointer-events-none relative z-10 mx-auto flex max-w-5xl flex-col items-center px-6 pt-48 pb-36 text-center">
          <span className="mb-5 inline-block rounded-full border border-border bg-surface-raised/20 px-3 py-1 text-xs font-medium tracking-wide text-ink-secondary backdrop-blur-sm">
            Infrastructure, generated
          </span>
          <h1 className="max-w-2xl text-5xl font-semibold leading-tight tracking-tight text-ink">
            Go from idea to cloud&nbsp;infrastructure in minutes
          </h1>
          <p className="mt-5 max-w-xl text-lg leading-relaxed text-ink-secondary">
            Describe your project in plain English. GoMCP figures out the
            architecture, writes the Terraform, and gets you ready to deploy —
            no cloud expertise required.
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            { isAuthed ? (

              <Link
                to="/projects"
                className="pointer-events-auto rounded-lg bg-brand px-6 py-2.5 text-sm font-medium text-white shadow-sm transition-colors hover:bg-brand-hover"
              >
                View my projects
              </Link>

              ) : (
              <>
                <Link
                  to="/signup"
                  className="pointer-events-auto rounded-lg bg-brand px-6 py-2.5 text-sm font-medium text-white shadow-sm transition-colors hover:bg-brand-hover"
                >
                  Start a project
                </Link>
                <Link
                  to="/login"
                  className="pointer-events-auto rounded-lg border border-border bg-surface-raised/30 px-6 py-2.5 text-sm font-medium text-ink-secondary shadow-sm backdrop-blur-sm transition-colors hover:bg-surface-raised/50"
                >
                  Sign in
                </Link>
              </>
            )}

          </div>
        </div>
      </section>

      {/* Divider */}
      <div className="mx-auto max-w-5xl px-6">
        <div className="h-px bg-border" />
      </div>

      {/* Features */}
      <section className="mx-auto max-w-5xl px-6 py-20">
        <h2 className="mb-12 text-center text-sm font-medium uppercase tracking-widest text-ink-muted">
          How it works
        </h2>
        <div className="grid gap-6 sm:grid-cols-3">
          {FEATURES.map((f, i) => (
            <div
              key={i}
              className="rounded-2xl border border-border bg-surface-raised px-6 py-7 shadow-sm"
            >
              <span className="mb-4 flex h-8 w-8 items-center justify-center rounded-lg bg-border-subtle text-sm font-semibold text-ink-secondary">
                {i + 1}
              </span>
              <h3 className="mb-2 text-base font-semibold text-ink">
                {f.title}
              </h3>
              <p className="text-sm leading-relaxed text-ink-secondary">
                {f.description}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* CTA band */}
      <section className="border-t border-border bg-surface-raised">
        <div className="mx-auto flex max-w-5xl flex-col items-center px-6 py-16 text-center">
          <h2 className="text-2xl font-semibold tracking-tight text-ink">
            Ready to build something?
          </h2>
          <p className="mt-3 text-sm text-ink-secondary">
            Create a free account and generate your first project in under five
            minutes.
          </p>
          <Link
            to={isAuthed ? "/projects" : "/signup"}
            className="mt-6 rounded-lg bg-brand px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover"
          >
            {isAuthed ? "Go to my projects" : "Get started for free"}
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border bg-bg">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-6">
          <span className="text-sm font-medium text-ink-secondary">GoMCP</span>
          <span className="text-xs text-ink-muted">
            &copy; {new Date().getFullYear()}
          </span>
        </div>
      </footer>
    </div>
  );
}

