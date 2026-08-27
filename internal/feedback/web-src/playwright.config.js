import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  timeout: 30_000,
  use: {
    baseURL: "http://127.0.0.1:41873",
    headless: true,
  },
  webServer: {
    // Exercise the committed single-file bundle served by Go in production,
    // not Vite's source-transforming dev server.
    command: "npm run preview -- --host 127.0.0.1 --port 41873",
    url: "http://127.0.0.1:41873",
    reuseExistingServer: false,
  },
});
