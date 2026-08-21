import { expect, test, type Page } from "@playwright/test";

const validPassword = "x".repeat(16);

function uniqueEmail(): string {
  return `m2-e2e-${crypto.randomUUID()}@example.test`;
}

function uniquePortfolioName(): string {
  return `M2 E2E Portfolio ${crypto.randomUUID()}`;
}

async function register(page: Page, email: string): Promise<void> {
  await page.goto("/register");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(validPassword);
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page).toHaveURL(/\/app$/);
  await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
}

async function createPortfolio(page: Page, name: string): Promise<string> {
  await page.goto("/app/portfolios");
  await page.getByLabel("Portfolio name").fill(name);
  await page.getByRole("button", { name: "Create Portfolio" }).click();

  await expect(page).toHaveURL(/\/app\/portfolios\/[^/]+$/);
  await expect(page.getByRole("heading", { name })).toBeVisible();
  return page.url();
}

async function assertAssetCatalog(page: Page): Promise<void> {
  await page.goto("/app/assets");
  const list = page.getByRole("list", { name: "Assets" });
  await expect(list).toBeVisible();

  await expect(list.getByText("M2A01", { exact: true })).toBeVisible();
  await expect(
    list.getByText("Synthetic Equity Alpha", { exact: true }),
  ).toBeVisible();
  await expect(list.getByText("M2A02", { exact: true })).toBeVisible();
  await expect(list.getByText("M2A03", { exact: true })).toBeVisible();

  const cryptoAsset = list.getByRole("listitem").filter({ hasText: "M2A03" });
  await expect(cryptoAsset).toContainText("Synthetic Crypto Gamma");
  const exchangeMetadata = cryptoAsset
    .locator("dl > div")
    .filter({ hasText: "Exchange" });
  await expect(
    exchangeMetadata.getByText("CRYPTO", { exact: true }),
  ).toBeVisible();
  const currencyMetadata = cryptoAsset
    .locator("dl > div")
    .filter({ hasText: "Currency" });
  await expect(
    currencyMetadata.getByText("USD", { exact: true }),
  ).toBeVisible();
  await expect(list.getByText(/COINBASE|BINANCE|KRAKEN/i)).toHaveCount(0);

  await expect(page.getByRole("button", { name: "Load more" })).toBeVisible();
  await expect(list.getByText("M2Z99", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Load more" }).click();
  await expect(list.getByText("M2Z99", { exact: true })).toBeVisible();
  await expect(list.getByText("M2A01", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Load more" })).toHaveCount(0);

  await page.getByLabel("Search assets").fill("Synthetic ETF Beta");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect(list.getByText("M2A02", { exact: true })).toBeVisible();
  await expect(list.getByText("M2A01", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Clear search" }).click();

  await page.getByRole("button", { name: "Crypto" }).click();
  await expect(list.getByText("M2A03", { exact: true })).toBeVisible();
  await expect(list.getByText("M2A01", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Equity" }).click();
  await expect(list.getByText("M2A01", { exact: true })).toBeVisible();
  await expect(list.getByText("M2A03", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "ETF" }).click();
  await expect(list.getByText("M2A02", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "All" }).click();

  for (const unavailableControl of [
    "Add Asset",
    "Create Asset",
    "Edit Asset",
    "Delete Asset",
    "Import Asset",
    "Sync Asset",
    "Add to Portfolio",
    "Buy",
    "Sell",
  ]) {
    await expect(
      page.getByRole("button", { name: unavailableControl, exact: true }),
    ).toHaveCount(0);
  }

  await expect(
    page.getByText(
      /Market Value|Cost Basis|Average Cost|P\/L|Allocation|Returns|Portfolio Value/,
    ),
  ).toHaveCount(0);
}

test.describe.serial("M2 Portfolio and Asset real-stack critical flow", () => {
  test("proves portfolio lifecycle, ownership isolation, and Asset discovery", async ({
    page,
    context,
  }) => {
    const userAEmail = uniqueEmail();
    const userBEmail = uniqueEmail();
    const originalPortfolioName = uniquePortfolioName();
    const renamedPortfolioName = `${originalPortfolioName} Renamed`;
    const externalRequests: string[] = [];
    page.on("request", (request) => externalRequests.push(request.url()));

    await register(page, userAEmail);
    await expect(
      page.getByRole("link", { name: "Portfolios", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("link", { name: "Assets", exact: true }),
    ).toBeVisible();

    const archivedPortfolioURL = await createPortfolio(
      page,
      originalPortfolioName,
    );
    await expect(page.getByText("ACTIVE", { exact: true })).toBeVisible();
    await expect(page.getByText("Base currency: USD")).toBeVisible();
    await expect(
      page.getByText(
        /Market Value|Cost Basis|Average Cost|P\/L|Allocation|Returns|Portfolio Value/,
      ),
    ).toHaveCount(0);

    await page.getByRole("link", { name: "Back to Portfolios" }).click();
    await expect(
      page.getByRole("heading", { name: "Portfolios" }),
    ).toBeVisible();
    await page.getByLabel("Portfolio name").fill(originalPortfolioName);
    await page.getByRole("button", { name: "Create Portfolio" }).click();
    await expect(
      page.getByText("An active portfolio with this name already exists.", {
        exact: true,
      }),
    ).toBeVisible();

    await page.getByRole("link", { name: originalPortfolioName }).click();
    await expect(
      page.getByRole("heading", { name: originalPortfolioName }),
    ).toBeVisible();
    await page.waitForLoadState("networkidle");
    const renameInput = page.getByLabel("Portfolio name");
    await expect(renameInput).toHaveValue(originalPortfolioName);
    await renameInput.fill(renamedPortfolioName);
    await expect(renameInput).toHaveValue(renamedPortfolioName);
    const renameResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "PATCH" &&
        response.url().includes("/api/v1/portfolios/"),
    );
    await page.getByRole("button", { name: "Save name" }).click();
    const response = await renameResponse;
    expect(response.ok()).toBe(true);
    await expect(response.json()).resolves.toMatchObject({
      name: renamedPortfolioName,
    });
    await expect(
      page.getByRole("heading", { name: renamedPortfolioName }),
    ).toBeVisible();
    await expect(page.getByText("ACTIVE", { exact: true })).toBeVisible();
    await expect(page.getByText("Base currency: USD")).toBeVisible();

    await page.getByRole("button", { name: "Archive Portfolio" }).click();
    await expect(
      page.getByRole("heading", { name: "Archive this Portfolio?" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Archive Portfolio" }).click();
    await expect(
      page.getByRole("heading", { name: "Archived Portfolio" }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Rename Portfolio" }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Archive Portfolio" }),
    ).toHaveCount(0);
    await expect(page.getByRole("button", { name: /Delete/i })).toHaveCount(0);

    await page.getByRole("link", { name: "Back to Portfolios" }).click();
    await expect(
      page.getByRole("heading", { name: "Portfolios" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Archived" }).click();
    await expect(
      page.getByRole("link", { name: renamedPortfolioName }),
    ).toBeVisible();
    await expect(page.getByLabel("Portfolio name")).toHaveCount(0);
    await page.getByRole("button", { name: "Active" }).click();
    await expect(
      page.getByRole("link", { name: renamedPortfolioName }),
    ).toHaveCount(0);

    const userAActivePortfolioURL = await createPortfolio(
      page,
      renamedPortfolioName,
    );
    expect(userAActivePortfolioURL).not.toBe(archivedPortfolioURL);
    await expect(page.getByText("ACTIVE", { exact: true })).toBeVisible();

    const archivedPortfolioPath = new URL(archivedPortfolioURL).pathname;
    await page.goto("/app/portfolios");
    await expect(
      page.getByRole("heading", { name: "Portfolios" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Archived" }).click();
    const archivedPortfolioLink = page.getByRole("link", {
      name: renamedPortfolioName,
    });
    await expect(archivedPortfolioLink).toBeVisible();
    await expect(archivedPortfolioLink).toHaveAttribute(
      "href",
      archivedPortfolioPath,
    );

    await assertAssetCatalog(page);
    await page.getByRole("button", { name: "Sign out" }).click();
    await expect(page).toHaveURL(/\/login$/);

    await register(page, userBEmail);
    await page.goto("/app/portfolios");
    await expect(
      page.getByRole("link", { name: renamedPortfolioName }),
    ).toHaveCount(0);
    await page.goto(new URL(userAActivePortfolioURL).pathname);
    await expect(
      page.getByRole("heading", { name: "Portfolio not found." }),
    ).toBeVisible();
    await expect(page.getByText(userAEmail)).toHaveCount(0);

    await assertAssetCatalog(page);
    await expect(page.evaluate(() => document.cookie)).resolves.not.toContain(
      "pra_rt_v1",
    );
    const storage = await page.evaluate(() => ({
      local: Object.entries(localStorage),
      session: Object.entries(sessionStorage),
    }));
    expect(JSON.stringify(storage)).not.toContain(userAEmail);
    expect(JSON.stringify(storage)).not.toContain(userBEmail);
    expect(JSON.stringify(storage)).not.toMatch(/access[_-]?token/i);
    await expect(
      (await context.cookies()).find((cookie) => cookie.name === "pra_rt_v1"),
    ).toBeDefined();

    expect(
      externalRequests.some((url) =>
        /yahoo|alpha[ -]?vantage|polygon|coingecko|coinbase|binance|kraken/i.test(
          url,
        ),
      ),
    ).toBe(false);
  });
});
