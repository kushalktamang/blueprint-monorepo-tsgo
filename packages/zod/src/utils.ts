import { z } from "zod";

interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  totalPages: number;
}

const schemaWithPagination = <T>(schema: z.ZodType<T>): z.ZodType<PaginatedResponse<T>> =>
  z.object({
    data: z.array(schema),
    total: z.number(),
    page: z.number(),
    limit: z.number(),
    totalPages: z.number(),
  });

export type { PaginatedResponse };
export { schemaWithPagination };
