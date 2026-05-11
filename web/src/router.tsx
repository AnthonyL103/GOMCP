import { createBrowserRouter } from "react-router-dom";
import Layout from "./layouts/Layout";
import AuthGuard from "./auth/AuthGuard";
import LandingPage from "./pages/LandingPage";
import ChatPage from "./pages/ChatPage";
import SignUpPage from "./pages/SignUpPage";
import VerifyPage from "./pages/VerifyPage";
import LoginPage from "./pages/LoginPage";
import ProjectsPage from "./pages/ProjectsPage";
import NewProjectPage from "./pages/NewProjectPage";

export const router = createBrowserRouter([
  // Unprotected routes — keep existing chat flow working
  {
    path: "/",
    element: <Layout />,
    children: [
      { index: true, element: <LandingPage /> },
      { path: "chat", element: <ChatPage /> },
    ],
  },
  
  // Auth pages — no sidebar
  { path: "/signup", element: <SignUpPage /> },
  { path: "/verify", element: <VerifyPage /> },
  { path: "/login", element: <LoginPage /> },
  
  // Protected routes — session required
  {
    element: <AuthGuard />,
    children: [
      {
        element: <Layout />,
        children: [
          { path: "/projects", element: <ProjectsPage /> },
          { path: "/projects/new", element: <NewProjectPage /> },
        ],
      },
    ],
  },
]);
