import { expect, test } from "@playwright/test";

test("application foundation starts", async ({ page }) => {
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Portfolio Research Assistant" }),
  ).toBeVisible();
});
