import { expect, test } from "@playwright/test";

const user = {
  id: "user_e2e",
  email: "person@example.com",
  status: "active",
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};

function accessTokenResponse() {
  return {
    accessToken: "browser-memory-value",
    tokenType: "Bearer",
    expiresIn: 900,
  };
}

async function mockAuthentication(page: import("@playwright/test").Page) {
  let serverSessionActive = false;
  await page.route("**/api/v1/auth/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path.endsWith("/register") || path.endsWith("/login")) {
      serverSessionActive = true;
      await route.fulfill({ json: { ...accessTokenResponse(), user } });
      return;
    }
    if (path.endsWith("/refresh")) {
      if (serverSessionActive) {
        await route.fulfill({ json: accessTokenResponse() });
      } else {
        await route.fulfill({
          status: 401,
          json: {
            error: {
              code: "SESSION_REFRESH_REJECTED",
              message: "Authentication required",
              correlationId: "e2e-correlation",
            },
          },
        });
      }
      return;
    }
    if (path.endsWith("/me")) {
      await route.fulfill({ json: user });
      return;
    }
    if (path.endsWith("/logout")) {
      serverSessionActive = false;
      await route.fulfill({ status: 204 });
      return;
    }
    await route.abort();
  });
}

test("application foundation starts", async ({ page }) => {
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Portfolio Research Assistant" }),
  ).toBeVisible();
});

test("unauthenticated protected navigation redirects to login", async ({
  page,
}) => {
  await mockAuthentication(page);
  await page.goto("/app");
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});

test("login restores an in-memory session after reload and logout invalidates recovery", async ({
  page,
}) => {
  await mockAuthentication(page);
  await page.goto("/login");
  await page.getByLabel("Email").fill("person@example.com");
  await page.getByLabel("Password").fill("x".repeat(12));
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(/\/app$/);
  await expect(page.getByText("Signed in as person@example.com")).toBeVisible();

  await page.reload();
  await expect(page.getByText("Signed in as person@example.com")).toBeVisible();
  const storage = await page.evaluate(() => ({
    local: Object.entries(localStorage),
    session: Object.entries(sessionStorage),
  }));
  expect(JSON.stringify(storage)).not.toContain("browser-memory-value");
  expect(JSON.stringify(storage)).not.toContain("person@example.com");

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login$/);
  await page.reload();
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});

test("registration establishes the protected memory session", async ({
  page,
}) => {
  await mockAuthentication(page);
  await page.goto("/register");
  await page.getByLabel("Email").fill("person@example.com");
  await page.getByLabel("Password").fill("x".repeat(12));
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/app$/);
  await expect(page.getByText("Signed in as person@example.com")).toBeVisible();
});
