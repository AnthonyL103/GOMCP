import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { listProjects, type Project } from "../api";

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      try {
        const items = await listProjects();
        if (!cancelled) {
          setProjects(items);
          setError(null);
        }
      } catch (err: unknown) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load projects");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="flex flex-1 flex-col px-6 py-10">
      <div className="mx-auto w-full max-w-5xl">
        <div className="flex items-center justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-ink-muted">
              Your projects
            </p>
            <h1 className="mt-2 text-2xl font-semibold tracking-tight text-ink">
              Saved sessions for your account
            </h1>
            <p className="mt-2 text-sm leading-relaxed text-ink-secondary">
              These are loaded from the backend by user id and can be reopened in the chat flow.
            </p>
          </div>
          <Link
            to="/projects/new"
            className="inline-flex rounded-lg bg-brand px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover"
          >
            Start a new project
          </Link>
        </div>

        <div className="mt-8 rounded-3xl border border-border bg-surface-raised p-6 shadow-sm">
          {loading ? (
            <div className="flex items-center gap-3 text-sm text-ink-secondary">
              <span className="inline-block h-3 w-3 animate-pulse rounded-full bg-brand" />
              Loading your projects…
            </div>
          ) : error ? (
            <div className="rounded-xl border border-red-500/20 bg-red-500/5 px-4 py-3 text-sm text-red-600">
              {error}
            </div>
          ) : projects.length === 0 ? (
            <div className="text-center">
              <h2 className="text-lg font-semibold text-ink">No saved projects yet</h2>
              <p className="mt-2 text-sm text-ink-secondary">
                Create a project, chat with the agent, and save the session to see it here.
              </p>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {projects.map((project) => (
                <Link
                  key={project.id}
                  to={`/projects/${project.id}/chat`}
                  className="rounded-2xl border border-border bg-bg p-5 text-left transition-transform hover:-translate-y-0.5 hover:shadow-md"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h2 className="text-base font-semibold text-ink">{project.name}</h2>
                      <p className="mt-1 text-xs text-ink-muted">
                        {project.id}
                      </p>
                    </div>
                    <span className="rounded-full bg-brand/10 px-2.5 py-1 text-xs font-medium text-brand">
                      {project.status}
                    </span>
                  </div>

                  <p className="mt-4 line-clamp-3 text-sm leading-relaxed text-ink-secondary">
                    {typeof project.answers?.description === "string"
                      ? project.answers.description
                      : "Open to continue the chat and review the generated Terraform."}
                  </p>

                  <div className="mt-4 flex items-center justify-between text-xs text-ink-muted">
                    <span>{project.user_id ?? "unknown user"}</span>
                    <span>{new Date(project.updated_at).toLocaleString()}</span>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}