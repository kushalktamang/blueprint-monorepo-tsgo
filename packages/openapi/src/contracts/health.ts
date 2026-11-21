import { ZHealthResponse } from "@blueprint/zod";
import { initContract } from "@ts-rest/core";

// import { z } from "zod";
// import getSecurityMetadata from "../utils.ts";

const c = initContract();

const healthContract = c.router({
  getHealth: {
    summary: "Get health",
    path: "/status",
    method: "GET",
    description: "Get health status",
    responses: {
      200: ZHealthResponse,
    },
  },
});

export default healthContract;
