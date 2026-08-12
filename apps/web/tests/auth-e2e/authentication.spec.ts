import { expect, test } from "@playwright/test";

function uniqueEmail() {
  return `auth-e2e-${crypto.randomUUID()}@example.test`;
}

async function register(page: import("@playwright/test").Page, email: string) {
  await page.goto("/register");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("x".repeat(16));
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/app$/);
  await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
}

async function expectSecureRefreshCookie(
  context: import("@playwright/test").BrowserContext,
) {
  const cookie = (await context.cookies()).find(
    (candidate) => candidate.name === "pra_rt_v1",
  );
  expect(cookie).toBeDefined();
  expect(cookie?.secure).toBe(true);
  expect(cookie?.httpOnly).toBe(true);
  expect(cookie?.sameSite).toBe("Lax");
  expect(cookie?.path).toBe("/api/v1/auth");
  expect(cookie?.domain).not.toMatch(/^\./);
}

test("real HTTPS registration reload recovery and logout", async ({
  page,
  context,
}) => {
  const email = uniqueEmail();
  await register(page, email);
  await expectSecureRefreshCookie(context);

  await expect(page.evaluate(() => document.cookie)).resolves.not.toContain(
    "pra_rt_v1",
  );
  const browserStorage = await page.evaluate(() => ({
    local: Object.entries(localStorage),
    session: Object.entries(sessionStorage),
  }));
  expect(JSON.stringify(browserStorage)).not.toContain(email);

  await page.reload();
  await expect(page).toHaveURL(/\/app$/);
  await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
  await expectSecureRefreshCookie(context);

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login$/);
  expect(
    (await context.cookies()).find((item) => item.name === "pra_rt_v1"),
  ).toBeUndefined();

  await page.reload();
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});

test("unauthenticated visitors never see the protected shell", async ({
  page,
}) => {
  await page.goto("/app");
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByText("Signed in as")).toHaveCount(0);
});

test("public Authentication pages redirect an active server session to the application", async ({
  page,
}) => {
  await register(page, uniqueEmail());
  await page.goto("/login");
  await expect(page).toHaveURL(/\/app$/);
  await page.goto("/register");
  await expect(page).toHaveURL(/\/app$/);
});
