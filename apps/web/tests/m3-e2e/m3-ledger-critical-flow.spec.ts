import { expect, test, type Locator, type Page } from "@playwright/test";
import type { components } from "@portfolio/api-contracts";
import { realCommand } from "./real-api";

type Transaction = components["schemas"]["Transaction"];
type Correction = components["schemas"]["TransactionCorrectionResult"];
const equityId = "30000000-0000-4000-8000-000000000001";
const cryptoId = "30000000-0000-4000-8000-000000000003";
const effectiveAt = "2026-01-02T15:30:00Z";
const keyGrammar = /^[A-Za-z0-9][A-Za-z0-9._~-]{15,127}$/;

function responseFor(page: Page, method: string, path: string) {
  return page.waitForResponse(
    (response) =>
      response.request().method() === method &&
      new URL(response.url()).pathname === path,
  );
}

async function register(page: Page, email: string) {
  await page.goto("/register");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("x".repeat(16));
  const registered = responseFor(page, "POST", "/api/v1/auth/register");
  await page.getByRole("button", { name: "Create account" }).click();
  expect((await registered).status()).toBe(201);
  await expect(page).toHaveURL(/\/app$/);
  await expect(page.getByText(`Signed in as ${email}`)).toBeVisible();
}

async function searchPicker(page: Page, form: Locator, symbol: string) {
  await form.getByLabel("Search Assets", { exact: true }).fill(symbol);
  const searched = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return (
      response.request().method() === "GET" &&
      url.pathname === "/api/v1/assets" &&
      url.searchParams.get("search") === symbol
    );
  });
  await form.getByRole("button", { name: "Search", exact: true }).click();
  expect((await searched).status()).toBe(200);
}

async function fillTrade(page: Page, form: Locator, quantity: string) {
  await searchPicker(page, form, "M3EQ01");
  await form
    .getByRole("button", {
      name: "M3EQ01 — Synthetic M3 Equity · EQUITY · NYSE · USD",
      exact: true,
    })
    .click();
  await form.getByLabel("Quantity", { exact: true }).fill(quantity);
  await form.getByLabel("Unit price (USD)", { exact: true }).fill("25.50");
  await form.getByLabel("Fee (USD, optional; defaults to zero)").fill("1.25");
  await form
    .getByLabel("Effective time (UTC)", { exact: true })
    .fill(effectiveAt);
}

async function assertFacts(row: Locator, transaction: Transaction) {
  for (const [label, value] of [
    ["Kind", transaction.kind],
    ["Asset ID", transaction.assetId],
    ["Quantity", transaction.quantity],
    ["Unit price", transaction.unitPrice],
    ["Fee", transaction.fee],
    ["Currency", transaction.currency],
    ["Effective time (UTC)", transaction.effectiveAt],
  ]) {
    // The existing public facts UI uses semantic dt/dd pairs.
    const pair = row.locator("dl > div").filter({
      has: row.page().getByText(label!, { exact: true }),
    });
    await expect(pair.locator("dd")).toHaveText(value!);
  }
}

async function assertNoForbiddenUI(page: Page) {
  for (const role of ["button", "link", "heading"] as const) {
    await expect(
      page.getByRole(role, {
        name: /^(Edit(?: Transaction)?|Delete(?: Transaction)?|Market Value|Portfolio Value|Holdings|Cost Basis|Average Cost|P\/L|Gain\/Loss|Allocation|Cash Balance|Returns)$/i,
      }),
    ).toHaveCount(0);
  }
  await expect(
    page.locator("dt").filter({
      hasText:
        /^(Market Value|Portfolio Value|Holdings|Cost Basis|Average Cost|P\/L|Gain\/Loss|Allocation|Cash Balance|Returns)$/i,
    }),
  ).toHaveCount(0);
}

async function assertStorageSafe(page: Page) {
  // Return boolean evidence only, never raw storage/cookie values to assertions.
  const evidence = await page.evaluate(() => ({
    refreshReadable: document.cookie.includes("pra_rt_v1"),
    credentialsPersisted: [localStorage, sessionStorage].some((storage) =>
      Object.entries(storage).some(([key, value]) =>
        /access[_-]?token|refresh[_-]?token|pra_rt_v1|Bearer |eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\./i.test(
          `${key} ${value}`,
        ),
      ),
    ),
  }));
  expect(evidence).toEqual({
    refreshReadable: false,
    credentialsPersisted: false,
  });
}

test("real ledger create, replay, correction, eligibility and owner isolation", async ({
  page,
  context,
}) => {
  const userA = `m3-e2e-${crypto.randomUUID()}@example.test`;
  const portfolioName = `M3 synthetic ledger ${crypto.randomUUID()}`;
  let externalProviderRequest = false;
  context.on("request", (request) => {
    if (
      /yahoo|alphavantage|alpha-vantage|polygon|coingecko|coinbase|binance|kraken|twelvedata|tiingo/i.test(
        new URL(request.url()).hostname,
      )
    )
      externalProviderRequest = true;
  });

  await test.step("real registration, catalog and owned Portfolio", async () => {
    await register(page, userA);
    await page.getByRole("link", { name: "Assets", exact: true }).click();
    await page.getByLabel("Search assets", { exact: true }).fill("M3CR01");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    const catalog = page.getByRole("list", { name: "Assets" });
    await expect(
      catalog.getByText("Synthetic M3 Crypto", { exact: true }),
    ).toBeVisible();
    const card = catalog.getByRole("listitem").filter({ hasText: "M3CR01" });
    await expect(
      card.locator("dl > div").filter({ hasText: "Exchange" }).locator("dd"),
    ).toHaveText("CRYPTO");
    await page.getByRole("link", { name: "Portfolios", exact: true }).click();
    await page.getByLabel("Portfolio name").fill(portfolioName);
  });

  const portfolioResponse = responseFor(page, "POST", "/api/v1/portfolios");
  await page.getByRole("button", { name: "Create Portfolio" }).click();
  const portfolioResult = await portfolioResponse;
  expect(portfolioResult.status()).toBe(201);
  const portfolio = (await portfolioResult.json()) as { id: string };
  const detailPath = `/app/portfolios/${portfolio.id}`;
  const ledgerPath = `${detailPath}/transactions`;
  const apiPath = `/api/v1/portfolios/${portfolio.id}/transactions`;
  await expect(page).toHaveURL((url) => url.pathname === detailPath);
  await expect(
    page.getByRole("heading", { name: portfolioName }),
  ).toBeVisible();
  await page.getByRole("link", { name: "Transactions", exact: true }).click();
  await expect(page).toHaveURL((url) => url.pathname === ledgerPath);
  await expect(
    page.getByText(`${portfolioName} · ACTIVE`, { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Record a Transaction" }),
  ).toBeVisible();
  const history = page.getByRole("region", { name: "Ledger history" });
  await expect(
    history.getByText("No transactions match these filters."),
  ).toBeVisible();
  await assertNoForbiddenUI(page);

  const form = page.getByRole("form", {
    name: "Transaction entry",
    exact: true,
  });
  await expect(
    form.getByLabel("Transaction kind").getByRole("option"),
  ).toHaveText(["BUY", "SELL", "DIVIDEND", "DEPOSIT", "WITHDRAWAL", "FEE"]);
  await expect(form.getByLabel("Asset type").getByRole("option")).toHaveText([
    "EQUITY",
    "ETF",
  ]);
  await searchPicker(page, form, "M3CR01");
  await expect(
    form.getByText(
      "No eligible Assets in these results. Try another search or load more.",
    ),
  ).toBeVisible();
  await expect(form.getByRole("button", { name: /M3CR01/ })).toHaveCount(0);
  await form.getByLabel("Asset type").selectOption("ETF");
  await searchPicker(page, form, "M3ETF01");
  await expect(
    form.getByRole("button", {
      name: "M3ETF01 — Synthetic M3 ETF · ETF · NYSEARCA · USD",
      exact: true,
    }),
  ).toBeVisible();
  await form.getByLabel("Asset type").selectOption("EQUITY");
  await fillTrade(page, form, "10");
  await form.getByRole("button", { name: "Review entry", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Review entry before submitting" }),
  ).toBeVisible();
  const createdResponse = responseFor(page, "POST", apiPath);
  await page
    .getByRole("button", { name: "Confirm submission", exact: true })
    .click();
  const created = await createdResponse;
  expect(created.status()).toBe(201);
  const command = {
    kind: "BUY",
    assetId: equityId,
    quantity: "10",
    unitPrice: "25.50",
    fee: "1.25",
    currency: "USD",
    effectiveAt,
  };
  expect(created.request().postDataJSON()).toEqual(command);
  const headers = created.request().headers();
  const createKey = headers["idempotency-key"];
  expect(createKey).toMatch(keyGrammar);
  expect(Boolean(headers.authorization?.startsWith("Bearer "))).toBe(true);
  const original = (await created.json()) as Transaction;
  expect(original).toMatchObject({ ...command, unitPrice: "25.5" });

  await test.step("real same-key replay creates no second fact", async () => {
    const replay = await realCommand(
      created.url(),
      headers.authorization,
      createKey,
      command,
    );
    expect(replay.status).toBe(201);
    expect(replay.body).toEqual(original);
    const reloaded = responseFor(page, "GET", apiPath);
    await page.reload();
    expect((await reloaded).status()).toBe(200);
    await expect(history.getByRole("heading", { level: 3 })).toHaveCount(1);
  });
  const originalRow = history.getByRole("listitem").filter({
    has: page.getByRole("heading", {
      name: `BUY · ${original.id}`,
      exact: true,
    }),
  });
  await assertFacts(originalRow, original);

  await test.step("real backend rejects catalog-only CRYPTO", async () => {
    const rejected = await realCommand(
      created.url(),
      headers.authorization,
      crypto.randomUUID(),
      { ...command, assetId: cryptoId },
    );
    expect(rejected.status).toBe(422);
    expect(rejected.body).toMatchObject({
      error: { code: "ASSET_FINANCIALLY_INELIGIBLE" },
    });
    const reloaded = responseFor(page, "GET", apiPath);
    await page.reload();
    const result = await reloaded;
    expect(result.status()).toBe(200);
    await expect(result.json()).resolves.toEqual({
      items: [original],
      nextCursor: null,
    });
    await expect(history.getByRole("heading", { level: 3 })).toHaveCount(1);
    await expect(history.getByText(cryptoId, { exact: true })).toHaveCount(0);
  });

  await originalRow
    .getByRole("button", { name: "Correct", exact: true })
    .click();
  const correction = page.getByRole("region", {
    name: "Correct Transaction",
    exact: true,
  });
  await expect(
    correction.getByRole("heading", { name: "Original — remains in history" }),
  ).toBeVisible();
  await expect(
    correction.getByText(original.id, { exact: true }),
  ).toBeVisible();
  await fillTrade(
    page,
    correction.getByRole("form", { name: "Complete replacement", exact: true }),
    "12",
  );
  await correction
    .getByRole("button", { name: "Review correction", exact: true })
    .click();
  const correctedResponse = responseFor(
    page,
    "POST",
    `${apiPath}/${original.id}/corrections`,
  );
  await correction
    .getByRole("button", { name: "Confirm correction", exact: true })
    .click();
  const corrected = await correctedResponse;
  expect(corrected.status()).toBe(201);
  const correctionKey = corrected.request().headers()["idempotency-key"];
  expect(correctionKey).toMatch(keyGrammar);
  expect(correctionKey).not.toBe(createKey);
  expect(corrected.request().postDataJSON()).toEqual({
    replacement: { ...command, quantity: "12" },
  });
  const result = (await corrected.json()) as Correction;
  expect(result.original.id).toBe(original.id);
  expect(result.original.correctionLinks).toMatchObject({
    reversalTransactionId: result.reversal.id,
    replacementTransactionId: result.replacement.id,
  });
  expect(result.reversal.correctionLinks.reversesTransactionId).toBe(
    original.id,
  );
  expect(result.replacement.correctionLinks.replacesTransactionId).toBe(
    original.id,
  );
  expect(result.reversal.kind).toBe("REVERSAL");
  expect(result.replacement).toMatchObject({
    ...command,
    quantity: "12",
    unitPrice: "25.5",
  });
  expect(
    new Set([result.original.id, result.reversal.id, result.replacement.id])
      .size,
  ).toBe(3);
  for (const [role, label] of [
    ["original", "Original — retained"],
    ["reversal", "Reversal — internal record"],
    ["replacement", "Replacement — new record"],
  ] as const) {
    const section = correction.getByRole("region", { name: role, exact: true });
    await expect(section.getByRole("heading", { name: label })).toBeVisible();
    await expect(
      section.getByText(result[role].id, { exact: true }),
    ).toBeVisible();
    await assertFacts(section, result[role]);
  }
  await correction.getByRole("button", { name: "Return to history" }).click();
  const recoveredHistory = responseFor(page, "GET", apiPath);
  await page.reload();
  const recovered = await recoveredHistory;
  expect(recovered.status()).toBe(200);
  await expect(recovered.json()).resolves.toEqual({
    items: [result.replacement, result.reversal, result.original],
    nextCursor: null,
  });
  await expect(history.getByRole("heading", { level: 3 })).toHaveCount(3);
  const rowFor = (transaction: Transaction) =>
    history.getByRole("listitem").filter({
      has: page.getByRole("heading", {
        name: `${transaction.kind} · ${transaction.id}`,
        exact: true,
      }),
    });
  await assertFacts(originalRow, original);
  await assertFacts(rowFor(result.reversal), result.reversal);
  await assertFacts(rowFor(result.replacement), result.replacement);
  for (const [source, label, target] of [
    [result.original, "Reversal", result.reversal],
    [result.original, "Replacement", result.replacement],
    [result.reversal, "Reverses", result.original],
    [result.replacement, "Replaces", result.original],
  ] as const) {
    const link = rowFor(source).getByRole("link", {
      name: `${label}: ${target.id}`,
      exact: true,
    });
    await expect(link).toHaveAttribute("href", `#transaction-${target.id}`);
    await link.click();
    await expect(rowFor(target)).toBeFocused();
  }
  await expect(
    originalRow.getByRole("button", { name: "Correct", exact: true }),
  ).toHaveCount(0);
  await expect(
    rowFor(result.reversal).getByRole("button", {
      name: "Correct",
      exact: true,
    }),
  ).toHaveCount(0);
  await expect(
    rowFor(result.replacement).getByRole("button", {
      name: "Correct",
      exact: true,
    }),
  ).toBeEnabled();
  await assertNoForbiddenUI(page);
  await assertStorageSafe(page);

  await test.step("another real user cannot see the Portfolio or ledger", async () => {
    await page.getByRole("button", { name: "Sign out", exact: true }).click();
    await expect(page).toHaveURL(/\/login$/);
    await register(page, `m3-e2e-${crypto.randomUUID()}@example.test`);
    for (const path of [detailPath, ledgerPath]) {
      const denied = responseFor(
        page,
        "GET",
        `/api/v1/portfolios/${portfolio.id}`,
      );
      await page.goto(path);
      const response = await denied;
      expect(response.status()).toBe(404);
      await expect(response.json()).resolves.toMatchObject({
        error: { code: "PORTFOLIO_NOT_FOUND" },
      });
      await expect(
        page.getByText("Portfolio not found.", { exact: true }),
      ).toBeVisible();
      for (const privateFact of [
        portfolioName,
        userA,
        original.id,
        result.reversal.id,
        result.replacement.id,
      ])
        await expect(page.getByText(privateFact, { exact: false })).toHaveCount(
          0,
        );
      await expect(
        page.getByRole("heading", { name: "Ledger history" }),
      ).toHaveCount(0);
    }
    await assertStorageSafe(page);
  });
  expect(
    externalProviderRequest,
    "No browser calls to market-data providers",
  ).toBe(false);
});
