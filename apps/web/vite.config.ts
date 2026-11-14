import path from "node:path";

import babel from "@rolldown/plugin-babel";
import tailwindcss from "@tailwindcss/vite";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import { defineConfig } from "vite-plus";
import type { PluginOption } from "vite-plus";
import { lazyPlugins } from "vite-plus";

export default defineConfig({
  plugins: lazyPlugins(() => [
    react(),
    babel({
      presets: [reactCompilerPreset()],
    }),
    tailwindcss() as PluginOption,
  ]),
  server: {
    port: 3000,
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@blueprint/openapi": path.resolve(__dirname, "../../packages/openapi/src"),
      "@blueprint/zod": path.resolve(__dirname, "../../packages/zod/src"),
    },
  },
});
