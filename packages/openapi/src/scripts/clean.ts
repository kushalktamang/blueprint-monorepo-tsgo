import { rm } from "node:fs/promises";

const targets = ["dist", "tsconfig.tsbuildinfo", ".turbo", "node_modules"];

await Promise.all(targets.map((target) => rm(target, { recursive: true, force: true })));
