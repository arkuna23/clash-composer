import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import "./i18n";
import { App } from "./App";
import { ThemeProvider } from "@/components/theme/ThemeProvider";

const root = document.getElementById("root");
if (!root) {
  throw new Error("missing #root container");
}

createRoot(root).render(
  <StrictMode>
    <ThemeProvider>
      <App />
    </ThemeProvider>
  </StrictMode>,
);
