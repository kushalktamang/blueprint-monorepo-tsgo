import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import "./index.css";
import App from "./app.tsx";

const rootElem = document.querySelector("root");
if (rootElem === null) {
  throw new Error("root element is not defined in the root html or is null");
}

createRoot(rootElem).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
