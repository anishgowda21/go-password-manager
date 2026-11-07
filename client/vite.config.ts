import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
    testTimeout: 60_000,
    hookTimeout: 60_000,
  },
  server: {
    proxy: { "/api": "http://localhost:8090" },
  },
});
