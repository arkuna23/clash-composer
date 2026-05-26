import { createBrowserRouter, Navigate } from "react-router-dom";
import { AppShell } from "@/components/layout/AppShell";
import { RequireAuth } from "@/components/layout/RequireAuth";
import { LoginPage } from "@/pages/LoginPage";
import { ConfigListPage } from "@/pages/ConfigListPage";
import { ConfigDetailPage } from "@/pages/ConfigDetailPage";

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      { path: "/", element: <Navigate to="/configs" replace /> },
      { path: "/configs", element: <ConfigListPage /> },
      { path: "/configs/:id", element: <ConfigDetailPage /> },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/configs" replace />,
  },
]);
