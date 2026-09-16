import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Dev server proxies /api to the Go backend (`go run ./cmd/serve`, default
// :8090) so `npm run dev` doesn't need CORS handling. In production the Go
// binary serves this app's build output directly (see cmd/serve/main.go's
// go:embed), so no proxy exists there — it's the same origin.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://localhost:8090",
    },
  },
  build: {
    outDir: "dist",
  },
});
