import fs from "fs";

import { OpenAPI } from "./index.js";

const replaceCustomFileTypesToOpenApiCompatible = (jsonString: string): string => {
  const searchPattern =
    /\{"type":"object","properties":\{"type":\{"type":"string","enum":\["file"\]\}\},\s*"required":\["type"\]\}/gu;
  const replacement = `{"type":"string","format":"binary"}`;

  return jsonString.replace(searchPattern, replacement);
};

const filteredDoc = replaceCustomFileTypesToOpenApiCompatible(JSON.stringify(OpenAPI));

const formattedDoc: unknown = JSON.parse(filteredDoc);

const filePaths = ["./openapi.json", "../../apps/server/internal/assets/static/openapi.json"];

filePaths.forEach((filePath) => {
  fs.writeFile(filePath, JSON.stringify(formattedDoc, null, 2), (err) => {
    if (err !== null) {
      throw err;
    }
  });
});
