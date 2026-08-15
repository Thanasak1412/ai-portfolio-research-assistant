import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

import { expect, test as setup } from "@playwright/test";

import { uniqueEmail, validPassword } from "./auth-fixtures";

const authDirectory = path.resolve("playwright/.auth");
const authMetadataPath = path.join(
  authDirectory,
  "generated-authenticated.json.meta",
);

setup("create the shared generated-suite account", async ({ page }) => {
  const email = uniqueEmail();
  await page.goto("/register");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(validPassword);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/app$/);
  await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();

  await mkdir(authDirectory, { recursive: true });
  await writeFile(authMetadataPath, JSON.stringify({ email }), "utf8");
});
