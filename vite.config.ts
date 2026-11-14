import fmtShared from "@blueprint/oxc/oxfmt";
import lintShared from "@blueprint/oxc/oxlint";
import { defineConfig } from "vite-plus";

export default defineConfig({
  fmt: {
    ...fmtShared,
    ignorePatterns: ["dist", "build", "node_modules", "**/dist/**", "**/build/**"],
  },
  lint: {
    extends: [lintShared],
    jsPlugins: [
      {
        name: "vite-plus",
        specifier: "vite-plus/oxlint-plugin",
      },
    ],
    rules: {
      "vite-plus/prefer-vite-plus-imports": "error",
    },
    ignorePatterns: ["dist", "build", "node_modules", "packages/httpStatus/src/**/*.ts"],
  },
  run: {
    cache: true,
  },
});
