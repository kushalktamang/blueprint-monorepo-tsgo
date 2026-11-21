import { expect, test } from "vite-plus/test";

import { OpenAPI } from "../src/index.ts";

test("generates the health OpenAPI document", () => {
  expect(OpenAPI.openapi).toBe("3.0.2");
  expect(OpenAPI.paths).toHaveProperty("/status");
});
