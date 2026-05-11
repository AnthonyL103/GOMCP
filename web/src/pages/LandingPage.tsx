import { Link } from "react-router-dom";

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
  return (
    <div className="min-h-screen bg-stone-50 text-stone-900">
      {/* Nav */}
      <header className="sticky top-0 z-10 border-b border-stone-200 bg-stone-50/80 backdrop-blur-sm">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-6">
          <span className="text-base font-semibold tracking-tight text-stone-800">
            GoMCP
          </span>
          <nav className="flex items-center gap-2">
            <Link
              to="/login"
              className="rounded-lg px-4 py-1.5 text-sm font-medium text-stone-600 transition-colors hover:bg-stone-100 hover:text-stone-800"
            >
              Sign in
            </Link>
            <Link
              to="/signup"
              className="rounded-lg bg-stone-800 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-stone-700"
            >
              Get started
            </Link>
          </nav>
        </div>
      </header>

      {/* Hero */}
      <section className="mx-auto flex max-w-5xl flex-col items-center px-6 pt-24 pb-20 text-center">
        <span className="mb-5 inline-block rounded-full border border-stone-200 bg-white px-3 py-1 text-xs font-medium tracking-wide text-stone-500">
          Infrastructure, generated
        </span>
        <h1 className="max-w-2xl text-5xl font-semibold leading-tight tracking-tight text-stone-800">
          Go from idea to cloud&nbsp;infrastructure in minutes
        </h1>
        <p className="mt-5 max-w-xl text-lg leading-relaxed text-stone-500">
          Describe your project in plain English. GoMCP figures out the
          architecture, writes the Terraform, and gets you ready to deploy —
          no cloud expertise required.
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <Link
            to="/signup"
            className="rounded-lg bg-stone-800 px-6 py-2.5 text-sm font-medium text-white shadow-sm transition-colors hover:bg-stone-700"
          >
            Start a project
          </Link>
          <Link
            to="/login"
            className="rounded-lg border border-stone-200 bg-white px-6 py-2.5 text-sm font-medium text-stone-700 shadow-sm transition-colors hover:bg-stone-50"
          >
            Sign in
          </Link>
        </div>
      </section>

      {/* Divider */}
      <div className="mx-auto max-w-5xl px-6">
        <div className="h-px bg-stone-200" />
      </div>

      {/* Features */}
      <section className="mx-auto max-w-5xl px-6 py-20">
        <h2 className="mb-12 text-center text-sm font-medium uppercase tracking-widest text-stone-400">
          How it works
        </h2>
        <div className="grid gap-6 sm:grid-cols-3">
          {FEATURES.map((f, i) => (
            <div
              key={i}
              className="rounded-2xl border border-stone-200 bg-white px-6 py-7 shadow-sm"
            >
              <span className="mb-4 flex h-8 w-8 items-center justify-center rounded-lg bg-stone-100 text-sm font-semibold text-stone-600">
                {i + 1}
              </span>
              <h3 className="mb-2 text-base font-semibold text-stone-800">
                {f.title}
              </h3>
              <p className="text-sm leading-relaxed text-stone-500">
                {f.description}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* CTA band */}
      <section className="border-t border-stone-200 bg-white">
        <div className="mx-auto flex max-w-5xl flex-col items-center px-6 py-16 text-center">
          <h2 className="text-2xl font-semibold tracking-tight text-stone-800">
            Ready to build something?
          </h2>
          <p className="mt-3 text-sm text-stone-500">
            Create a free account and generate your first project in under five
            minutes.
          </p>
          <Link
            to="/signup"
            className="mt-6 rounded-lg bg-stone-800 px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-stone-700"
          >
            Get started for free
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-stone-200 bg-stone-50">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-6">
          <span className="text-sm font-medium text-stone-500">GoMCP</span>
          <span className="text-xs text-stone-400">
            &copy; {new Date().getFullYear()}
          </span>
        </div>
      </footer>
    </div>
  );
}

