import { useEffect } from "react";
import { RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "sonner";
import { router } from "@/router";
import { setUnauthorizedHandler } from "@/api/client";
import { clearStoredToken } from "@/stores/auth";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
      staleTime: 30 * 1000,
    },
  },
});

export function App() {
  useEffect(() => {
    setUnauthorizedHandler(() => {
      clearStoredToken();
      // Navigation is handled by RequireAuth on the next render.
    });
    return () => setUnauthorizedHandler(null);
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <Toaster richColors closeButton position="top-right" />
    </QueryClientProvider>
  );
}
