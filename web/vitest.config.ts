import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "v8",
      reportsDirectory: "coverage",
      reporter: ["text", "text-summary", "json-summary", "html"],
      include: ["src/components/**", "src/lib/**", "src/app/**"],
      exclude: ["src/test/**", "src/**/*.test.{ts,tsx}"],
    },
  },
});
