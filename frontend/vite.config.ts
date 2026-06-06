import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

// Vite config — outputs to ../internal/embed/dist so the Go embed
// picks it up directly. Phase 1 ships with the bare minimum; later
// phases add proxy entries for /api, /auth, and the WebSocket.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: path.resolve(__dirname, '../internal/embed/dist'),
    emptyOutDir: true,
    sourcemap: true,
    target: 'es2022',
  },
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://localhost:8443', changeOrigin: true, secure: false },
      '/auth': { target: 'http://localhost:8443', changeOrigin: true, secure: false },
      '/healthz': { target: 'http://localhost:8443', changeOrigin: true, secure: false },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
});
