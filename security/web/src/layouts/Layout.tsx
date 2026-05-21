import { Outlet, Link, useNavigate } from "react-router-dom";
import { signOut } from "aws-amplify/auth";

export default function Layout() {
  const navigate = useNavigate();

  async function handleSignOut() {
    await signOut();
    void navigate("/login", { replace: true });
  }

  return (
    <div className="flex h-screen bg-stone-50 text-stone-900">
      {/* sidebar */}
      <aside className="hidden md:flex w-64 flex-col border-r border-stone-200 bg-stone-50">
        <div className="flex h-14 items-center px-4">
          <span className="text-lg font-semibold tracking-tight text-stone-800">
            GoMCP
          </span>
        </div>
        <div className="flex-1 overflow-y-auto px-3 py-2">
          <Link
            to="/projects/new"
            className="block w-full rounded-lg px-3 py-2 text-left text-sm text-stone-500 hover:bg-stone-100 transition-colors"
          >
            + New Project
          </Link>
        </div>
        <div className="border-t border-stone-200 px-3 py-3">
          <button
            onClick={handleSignOut}
            className="w-full rounded-lg px-3 py-2 text-left text-sm text-stone-400 hover:bg-stone-100 hover:text-stone-600 transition-colors"
          >
            Sign out
          </button>
        </div>
      </aside>

      {/* main area */}
      <div className="flex flex-1 flex-col min-w-0">
        <header className="flex h-14 shrink-0 items-center border-b border-stone-200 bg-white/80 px-4 backdrop-blur-sm">
          <span className="text-sm font-medium text-stone-600 md:hidden">
            GoMCP
          </span>
        </header>
        <Outlet />
      </div>
    </div>
  );
}
