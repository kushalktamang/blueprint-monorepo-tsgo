import { expect, test } from "vite-plus/test";
import { z } from "zod";

import { schemaWithPagination, ZHealthResponse } from "../src/index.ts";

test("validates a health response", () => {
  const response = {
    status: "healthy",
    timestamp: "2026-11-17T12:00:00.000Z",
    environment: "test",
    checks: {
      database: { status: "healthy", response_time: "5ms" },
    },
  };

  expect(ZHealthResponse.parse(response)).toEqual(response);
});

test("rejects an invalid health status and timestamp", () => {
  const result = ZHealthResponse.safeParse({
    status: "degraded",
    timestamp: "not-a-timestamp",
    environment: "test",
    checks: {
      database: { status: "healthy", response_time: "5ms" },
    },
  });

  expect(result.success).toBe(false);
});

test("validates paginated data against the provided item schema", () => {
  const responseSchema = schemaWithPagination(z.object({ id: z.number() }));
  const response = {
    data: [{ id: 1 }, { id: 2 }],
    total: 2,
    page: 1,
    limit: 10,
    totalPages: 1,
  };

  expect(responseSchema.parse(response)).toEqual(response);
  expect(responseSchema.safeParse({ ...response, data: [{ id: "1" }] }).success).toBe(false);
});
