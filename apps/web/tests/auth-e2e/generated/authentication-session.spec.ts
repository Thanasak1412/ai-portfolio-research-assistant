import { expect, test } from "@playwright/test";

import { login, logout } from "./auth-fixtures";

test.describe("Generated Authentication session coverage", () => {
  test.describe("with a fresh authenticated session", () => {
    test("recovers the authenticated session after a full reload", async ({
      page,
    }) => {
      const email = await login(page);
      await page.reload();
      await expect(page).toHaveURL(/\/app$/);
      await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
    });

    test("logs out and remains unauthenticated after reload and protected navigation", async ({
      page,
    }) => {
      await login(page);
      await logout(page);
      await page.reload();
      await expect(page).toHaveURL(/\/login$/);
      await page.goto("/app");
      await expect(page).toHaveURL(/\/login$/);
      await expect(page.getByText("Signed in as")).toHaveCount(0);
    });

    test("invalidates another same-context tab after logout", async ({
      page,
      context,
    }) => {
      await login(page);
      const secondPage = await context.newPage();
      await secondPage.goto("/app");
      await expect(secondPage.getByText(/Signed in as /)).toBeVisible();

      await logout(page);
      await expect(secondPage).toHaveURL(/\/login$/);
      await expect(secondPage.getByText("Signed in as")).toHaveCount(0);
      await secondPage.goto("/app");
      await expect(secondPage).toHaveURL(/\/login$/);
      await secondPage.close();
    });

    test("keeps browser authentication material out of script-visible storage", async ({
      page,
      context,
    }) => {
      await login(page);
      const cookie = (await context.cookies()).find(
        (candidate) => candidate.name === "pra_rt_v1",
      );
      expect(cookie).toMatchObject({
        secure: true,
        httpOnly: true,
        sameSite: "Lax",
        path: "/api/v1/auth",
      });
      await expect(page.evaluate(() => document.cookie)).resolves.not.toContain(
        "pra_rt_v1",
      );
      const storage = await page.evaluate(() => ({
        local: Object.keys(localStorage),
        session: Object.keys(sessionStorage),
      }));
      expect(JSON.stringify(storage)).not.toMatch(
        /accessToken|refreshToken|pra_rt_v1|auth-agent-/i,
      );
    });
  });
});
