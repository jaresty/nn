import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { viteSingleFile } from "vite-plugin-singlefile";

// Build the Excalidraw canvas surface into ../web as a single self-contained
// index.html (JS + CSS inlined), which internal/feedback/server.go serves via
// go:embed. Emitting one file keeps the embed simple and avoids asset-path
// coupling between the Vite output and the Go static route.
export default defineConfig({
  plugins: [react(), viteSingleFile()],
  define: {
    // The Excalidraw package entry is a CommonJS shim that switches bundles on
    // these env vars; define them so the production bundle is selected.
    "process.env.IS_PREACT": JSON.stringify("false"),
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
  build: {
    outDir: "../web",
    emptyOutDir: true,
    chunkSizeWarningLimit: 8000,
  },
});
