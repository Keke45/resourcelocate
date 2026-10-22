import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The Go server embeds the build from internal/webui/dist.
export default defineConfig({
  plugins: [react()],
  build: { outDir: "../internal/webui/dist", emptyOutDir: true },
  server: { proxy: { "/api": "http://localhost:8080" } },
});
