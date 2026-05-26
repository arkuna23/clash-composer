import { useSyncExternalStore } from "react";
import {
  clearStoredToken,
  getStoredToken,
  setStoredToken,
  subscribeToken,
} from "@/stores/auth";

export function useAuthToken(): {
  token: string | null;
  isAuthenticated: boolean;
  setToken: (token: string) => void;
  clearToken: () => void;
} {
  const token = useSyncExternalStore(
    subscribeToken,
    getStoredToken,
    () => null,
  );

  return {
    token,
    isAuthenticated: !!token,
    setToken: setStoredToken,
    clearToken: clearStoredToken,
  };
}
