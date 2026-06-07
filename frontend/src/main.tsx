/**
 * Entry point. The Vite-generated bundle inlines the bootstrap
 * script in index.html; this file is the React mount.
 */

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "@/App";
import "@/styles/globals.css";

const el = document.getElementById("root");
if (!el) throw new Error("missing #root element in index.html");

createRoot(el).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
