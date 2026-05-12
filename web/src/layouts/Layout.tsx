import { Outlet, Link, useNavigate } from "react-router-dom";
import { signOut } from "aws-amplify/auth";

export default function Layout() {
  const navigate = useNavigate();

  async function handleSignOut() {
    await signOut();
    void navigate("/", { replace: true });
  }

  return (
    <div className="flex h-screen bg-bg text-ink">
      {/* sidebar */}
      <aside className="hidden md:flex w-64 flex-col border-r border-border bg-bg">
        <div className="flex h-14 items-center px-4">
          <span className="text-lg font-semibold tracking-tight text-ink">
            GoMCP
          </span>
        </div>
        <div className="flex-1 overflow-y-auto px-3 py-2">
          <Link
            to="/projects/new"
            className="block w-full rounded-lg px-3 py-2 text-left text-sm text-ink-secondary hover:bg-border-subtle transition-colors"
          >
            + New Project
          </Link>
        </div>
        <div className="border-t border-border px-3 py-3">
          <button
            onClick={handleSignOut}
            className="w-full rounded-lg px-3 py-2 text-left text-sm text-ink-muted hover:bg-border-subtle hover:text-ink-secondary transition-colors"
          >
            Sign out
          </button>
        </div>
      </aside>

      {/* main area */}
      <div className="flex flex-1 flex-col min-w-0">
        <header className="flex h-14 shrink-0 items-center border-b border-border bg-surface-raised/80 px-4 backdrop-blur-sm">
          <span className="text-sm font-medium text-ink-secondary md:hidden">
            GoMCP
          </span>
        </header>
        <Outlet />
      </div>
    </div>
  );
}
