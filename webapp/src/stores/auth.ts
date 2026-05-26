// Lightweight auth store: token persisted in localStorage and exposed through
// a tiny pub/sub so the React tree can reactively show/hide the login screen.

const TOKEN_KEY = "clash-composer.token";

type Listener = (token: string | null) => void;
const listeners = new Set<Listener>();

export function getStoredToken(): string | null {
  try {
    return window.localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setStoredToken(token: string) {
  try {
    window.localStorage.setItem(TOKEN_KEY, token);
  } catch {
    // ignore: Safari private mode etc.
  }
  listeners.forEach((listener) => listener(token));
}

export function clearStoredToken() {
  try {
    window.localStorage.removeItem(TOKEN_KEY);
  } catch {
    // ignore
  }
  listeners.forEach((listener) => listener(null));
}

export function subscribeToken(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
