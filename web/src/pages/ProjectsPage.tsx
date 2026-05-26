import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { listProjects, type Project } from "../api";

function ProjectCard({ project }: { project: Project }) {
  const date = new Date(project.created_at).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });

  return (
    <Link
      to={`/projects/${project.id}/chat`}
      className="group flex flex-col rounded-2xl border border-border bg-surface-raised p-6 shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="flex-1">
        <h3 className="text-base font-semibold text-ink group-hover:text-ink-secondary transition-colors">
          {project.name}
        </h3>
        {project.answers?.description && (
          <p className="mt-1.5 text-sm leading-relaxed text-ink-secondary line-clamp-2">
            {project.answers.description}
          </p>
        )}
      </div>
      <div className="mt-4 flex items-center justify-between border-t border-border-subtle pt-4">
        <span className="text-xs text-ink-muted">{date}</span>
        <span className="text-xs font-medium text-ink-muted group-hover:text-ink-secondary transition-colors">
          View →
        </span>
      </div>
    </Link>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl border border-border bg-surface-raised shadow-sm">
        <svg
          className="h-6 w-6 text-ink-muted"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.5}
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M2.25 12.75V12A2.25 2.25 0 0 1 4.5 9.75h15A2.25 2.25 0 0 1 21.75 12v.75m-8.69-6.44-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v8.25"
          />
        </svg>
      </div>
      <h3 className="text-base font-semibold text-ink">No projects yet</h3>
      <p className="mt-1.5 max-w-xs text-sm text-ink-secondary">
        Create your first project and GoMCP will generate the infrastructure for
        you.
      </p>
      <Link
        to="/projects/new"
        className="mt-6 rounded-lg bg-brand px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover"
      >
        Create your first project
      </Link>
    </div>
  );
}

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listProjects()
      .then(setProjects)
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : "Failed to load projects");
      })
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="flex flex-1 flex-col overflow-y-auto">
      {/* Page header */}
      <div className="flex items-center justify-between border-b border-border bg-surface-raised/80 px-8 py-5 backdrop-blur-sm">
        <h1 className="text-xl font-semibold tracking-tight text-ink">
          Your Projects
        </h1>
        {projects.length > 0 && (
          <Link
            to="/projects/new"
            className="rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-hover"
          >
            + New Project
          </Link>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 px-8 py-8">
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <div className="flex items-center gap-1.5">
              <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-ink-muted" />
              <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-ink-muted [animation-delay:150ms]" />
              <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-ink-muted [animation-delay:300ms]" />
            </div>
          </div>
        ) : error ? (
          <div className="flex items-center justify-center py-24">
            <p className="text-sm text-red-500">{error}</p>
          </div>
        ) : projects.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {projects.map((p) => (
              <ProjectCard key={p.id} project={p} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

