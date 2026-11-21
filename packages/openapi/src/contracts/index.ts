import { initContract } from "@ts-rest/core";

import healthContract from "./health.js";

const c = initContract();

const apiContract = c.router({
  Health: healthContract,
});

export default apiContract;
