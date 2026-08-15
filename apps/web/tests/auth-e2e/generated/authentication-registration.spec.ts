import { expect, test } from "@playwright/test";

import {
  logout,
  register,
  generatedAccountEmail,
  uniqueEmail,
  validPassword,
} from "./auth-fixtures";

test.describe("Generated Authentication registration coverage", () => {
  test("starts from the seed as a usable unauthenticated visitor", async ({
    page,
  }) => {
    await page.goto("/register");
    await expect(
      page.getByRole("heading", { name: "Create account" }),
    ).toBeVisible();
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.getByLabel("Password")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Create account" }),
    ).toBeVisible();
    await expect(page.getByText("Signed in as")).toHaveCount(0);

    await page.getByRole("link", { name: "Sign in instead" }).click();
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
    await page.getByRole("link", { name: "Create an account" }).click();
    await expect(page).toHaveURL(/\/register$/);
  });

  test("shows accessible registration validation without sending a request", async ({
    page,
  }) => {
    await page.goto("/register");
    const registrationRequests: string[] = [];
    page.on("request", (request) => {
      if (request.url().includes("/api/v1/auth/register")) {
        registrationRequests.push(request.url());
      }
    });

    await page.getByRole("button", { name: "Create account" }).click();
    await expect(
      page.getByText("Email is required", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Password must be at least 12 characters", {
        exact: true,
      }),
    ).toBeVisible();

    await page.getByLabel("Email").fill("not-an-email");
    await page.getByLabel("Password").fill(validPassword);
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(
      page.getByText("Enter a valid email address", { exact: true }),
    ).toBeVisible();

    await page.getByLabel("Email").fill(uniqueEmail());
    await page.getByLabel("Password").fill("short");
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(
      page.getByText("Password must be at least 12 characters", {
        exact: true,
      }),
    ).toBeVisible();

    await page.getByLabel("Password").fill("😀".repeat(257));
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(
      page.getByText("Password must not exceed 1,024 bytes", { exact: true }),
    ).toBeVisible();
    expect(registrationRequests).toHaveLength(0);
  });

  test("registers a unique account and establishes the protected session", async ({
    page,
    context,
  }) => {
    const email = await register(page);
    const cookie = (await context.cookies()).find(
      (candidate) => candidate.name === "pra_rt_v1",
    );
    expect(cookie).toMatchObject({
      secure: true,
      httpOnly: true,
      sameSite: "Lax",
      path: "/api/v1/auth",
    });
    expect(cookie?.domain).not.toMatch(/^\./);
    await expect(page.evaluate(() => document.cookie)).resolves.not.toContain(
      "pra_rt_v1",
    );
    const browserStorage = await page.evaluate(() => ({
      local: Object.keys(localStorage),
      session: Object.keys(sessionStorage),
    }));
    expect(JSON.stringify(browserStorage)).not.toMatch(
      /accessToken|refreshToken|pra_rt_v1|auth-agent-/i,
    );
    expect(email).toContain("@example.test");
  });

  test("rejects duplicate registration without disclosing account existence", async ({
    page,
  }) => {
    const email = await generatedAccountEmail();
    await page.goto("/register");
    await page.getByLabel("Email").fill(email);
    await page.getByLabel("Password").fill(validPassword);
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(
      page.getByText("Registration could not be completed. Please try again.", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(page).toHaveURL(/\/register$/);
    await expect(page.getByLabel("Password")).toHaveValue("");
    await expect(page.getByText("Signed in as")).toHaveCount(0);
  });

  test("does not leave an authenticated session after registration validation fails", async ({
    page,
  }) => {
    await page.goto("/register");
    await page.getByLabel("Email").fill(uniqueEmail());
    await page.getByLabel("Password").fill("short");
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(page).toHaveURL(/\/register$/);
    await expect(page.getByText("Signed in as")).toHaveCount(0);
  });

  test.afterEach(async ({ page }) => {
    if (await page.getByRole("button", { name: "Sign out" }).count()) {
      await logout(page);
    }
  });
});
