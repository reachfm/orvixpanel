import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

// Vite config for the OrvixPanel Enterprise Admin UI.
//
// Build output goes to ./dist. The Go binary embeds that folder
// via go:embed and serves it under "/" (see internal/api/frontend.go
// in the v0.2.3 commit). When the embed is missing (e.g. fresh
// checkout, dev workflow) the binary falls back to serving the
// dist/ directory directly off disk.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    sourcemap: true,
  },
  server: {
    port: 5173,
    proxy: {
      // proxy API requests to the Go binary during dev.
      "/api":    { target: "http://127.0.0.1:28444", changeOrigin: true },
      "/auth":   { target: "http://127.0.0.1:28444", changeOrigin: true },
      "/healthz":{ target: "http://127.0.0.1:28444", changeOrigin: true },
      "/readyz": { target: "http://127.0.0.1:28444", changeOrigin: true },
    },
  },
});
