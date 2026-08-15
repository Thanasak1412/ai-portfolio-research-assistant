import { expect, test } from "@playwright/test";

import {
  generatedAccountEmail,
  uniqueEmail,
  validPassword,
} from "./auth-fixtures";

test.describe("Generated Authentication login coverage", () => {
  test("displays login controls and navigates to registration", async ({
    page,
  }) => {
    await page.goto("/login");
    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.getByLabel("Password")).toBeVisible();
    await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
    await page.getByRole("link", { name: "Create an account" }).click();
    await expect(page).toHaveURL(/\/register$/);
    await page.getByRole("link", { name: "Sign in instead" }).click();
    await expect(page).toHaveURL(/\/login$/);
  });

  test("shows accessible login validation without sending a request", async ({
    page,
  }) => {
    await page.goto("/login");
    const loginRequests: string[] = [];
    page.on("request", (request) => {
      if (request.url().includes("/api/v1/auth/login")) {
        loginRequests.push(request.url());
      }
    });

    await page.getByRole("button", { name: "Sign in" }).click();
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
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(
      page.getByText("Enter a valid email address", { exact: true }),
    ).toBeVisible();

    await page.getByLabel("Email").fill(uniqueEmail());
    await page.getByLabel("Password").fill("short");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(
      page.getByText("Password must be at least 12 characters", {
        exact: true,
      }),
    ).toBeVisible();
    expect(loginRequests).toHaveLength(0);
  });

  test("logs in with valid credentials through the real API", async ({
    page,
  }) => {
    const email = await generatedAccountEmail();
    await page.goto("/login");
    await page.getByLabel("Email").fill(email);
    await page.getByLabel("Password").fill(validPassword);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page).toHaveURL(/\/app$/);
    await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
  });

  test("rejects an unknown account with a generic authentication error", async ({
    page,
  }) => {
    await page.goto("/login");
    await page.getByLabel("Email").fill(uniqueEmail());
    await page.getByLabel("Password").fill(validPassword);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(
      page.getByText(
        "Authentication failed. Check your email and password and try again.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByLabel("Password")).toHaveValue("");
  });

  test("rejects an incorrect password without revealing account state", async ({
    page,
  }) => {
    const email = await generatedAccountEmail();
    await page.goto("/login");
    await page.getByLabel("Email").fill(email);
    await page.getByLabel("Password").fill("wrong-auth-agent-password");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(
      page.getByText(
        "Authentication failed. Check your email and password and try again.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByLabel("Password")).toHaveValue("");
  });
});
