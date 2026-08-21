import { expect, test } from "@playwright/test";

for (const path of ["/app", "/app/portfolios", "/app/assets"]) {
  test(`unauthenticated visitors cannot view ${path}`, async ({ page }) => {
    await page.goto(path);

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByText("Signed in as")).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Portfolios" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Assets" })).toHaveCount(0);
  });
}
