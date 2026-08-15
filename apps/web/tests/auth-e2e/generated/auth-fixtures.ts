import { readFile } from "node:fs/promises";
import path from "node:path";

import { expect, type Page } from "@playwright/test";

export function uniqueEmail(): string {
  return `auth-agent-${crypto.randomUUID()}@example.test`;
}

export const validPassword = "auth-agent-password";

export async function generatedAccountEmail(): Promise<string> {
  const metadata = JSON.parse(
    await readFile(
      path.resolve("playwright/.auth/generated-authenticated.json.meta"),
      "utf8",
    ),
  ) as { email?: unknown };
  if (typeof metadata.email !== "string" || !metadata.email) {
    throw new Error("Generated Authentication account metadata is invalid");
  }
  return metadata.email;
}

export async function register(
  page: Page,
  email = uniqueEmail(),
): Promise<string> {
  await page.goto("/register");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(validPassword);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/app$/);
  await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
  return email;
}

export async function logout(page: Page): Promise<void> {
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
}

export async function login(page: Page): Promise<string> {
  const email = await generatedAccountEmail();
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(validPassword);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(/\/app$/);
  await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
  return email;
}
