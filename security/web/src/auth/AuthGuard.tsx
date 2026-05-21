import { useEffect, useState } from "react";
import { Outlet, useNavigate } from "react-router-dom";
import { getCurrentUser } from "aws-amplify/auth";

export default function AuthGuard() {
  const [checking, setChecking] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    getCurrentUser()
      .then(() => setChecking(false))
      .catch(() => {
        void navigate("/login", { replace: true });
      });
  }, [navigate]);

  if (checking) {
    return (
      <div className="flex h-screen items-center justify-center bg-stone-50">
        <span className="text-sm text-stone-400">Loading…</span>
      </div>
    );
  }

  return <Outlet />;
}
