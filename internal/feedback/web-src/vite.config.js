import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { viteSingleFile } from "vite-plugin-singlefile";

// stripEmbeddedSecrets removes third-party API keys that the Excalidraw bundle
// inlines at its own build time (a public Firebase config for its hosted
// collaboration/room-persistence, which we do not use). The key is a literal in
// excalidraw.production.min.js, so a Vite `define` cannot reach it — we redact
// the emitted bundle instead. Verified: the canvas renders and edits fine
// without it. This keeps our committed go:embed bundle free of secrets.
function stripEmbeddedSecrets() {
  return {
    name: "strip-embedded-secrets",
    enforce: "post",
    generateBundle(_options, bundle) {
      const googleKey = /AIza[0-9A-Za-z_-]{35}/g;
      for (const file of Object.values(bundle)) {
        if (file.type === "asset" && typeof file.source === "string") {
          file.source = file.source.replace(googleKey, "");
        } else if (file.type === "chunk" && typeof file.code === "string") {
          file.code = file.code.replace(googleKey, "");
        }
      }
    },
  };
}

// Build the Excalidraw canvas surface into ../web as a single self-contained
// index.html (JS + CSS inlined), which internal/feedback/server.go serves via
// go:embed. Emitting one file keeps the embed simple and avoids asset-path
// coupling between the Vite output and the Go static route.
export default defineConfig({
  plugins: [react(), viteSingleFile(), stripEmbeddedSecrets()],
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
