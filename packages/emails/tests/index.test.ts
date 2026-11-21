import { createElement } from "react";
import { render } from "react-email";
import { expect, test } from "vite-plus/test";

import WelcomeEmail from "../src/templates/welcome.tsx";

test("renders a personalized welcome email", async () => {
  const html = await render(
    createElement(WelcomeEmail, {
      userFirstName: "Ada",
      baseUrl: "https://blueprint.example",
      year: 2026,
    }),
  );
  const normalizedHtml = html.replaceAll("<!-- -->", "");

  expect(normalizedHtml).toContain("Welcome to Blueprint");
  expect(normalizedHtml).toContain("Hi Ada,");
  expect(normalizedHtml).toContain("Thank you for joining!");
  expect(normalizedHtml).toContain('href="https://blueprint.example/dashboard"');
  expect(normalizedHtml).toContain('href="https://blueprint.example/support"');
  expect(normalizedHtml).toContain("© 2026 Blueprint. All rights reserved.");
});
