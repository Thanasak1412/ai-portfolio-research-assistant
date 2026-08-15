import { expect, test } from "@playwright/test";

import { login } from "./auth-fixtures";

test.describe("Generated Authentication routing coverage", () => {
  test("redirects unauthenticated /app without exposing protected content", async ({
    page,
  }) => {
    await page.goto("/app");
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByText("Signed in as")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Sign out" })).toHaveCount(0);
  });

  test.describe("with a fresh authenticated session", () => {
    test("redirects authenticated users from public auth routes to /app", async ({
      page,
    }) => {
      const email = await login(page);
      await expect(page).toHaveURL(/\/app$/);
      await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
      await page.goto("/register");
      await expect(page).toHaveURL(/\/app$/);
      await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
    });
  });
});
