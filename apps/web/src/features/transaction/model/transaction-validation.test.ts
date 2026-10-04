import { describe, expect, it } from "vitest";
import {
  commandSchema,
  correctionCommandSchema,
  createKinds,
  decimal,
  positiveDecimal,
  historyFiltersSchema,
  historyTime,
  isEligibleAsset,
} from "@/features/transaction/model/transaction-validation";
import { asset, command } from "@/features/transaction/test-fixtures";

describe("Transaction usability validation", () => {
  it("includes the correction wrapper in the byte limit and prohibits hidden target fields", () => {
    expect(correctionCommandSchema.parse({ replacement: command })).toEqual({
      replacement: command,
    });
    const base = { ...command, note: "😀".repeat(2000) };
    const bytes = new TextEncoder().encode(
      JSON.stringify({ ...base, externalReference: "" }),
    ).length;
    const nearLimit = { ...base, externalReference: "x".repeat(8192 - bytes) };
    expect(commandSchema.safeParse(nearLimit).success).toBe(true);
    expect(
      correctionCommandSchema.safeParse({ replacement: nearLimit }).success,
    ).toBe(false);
    expect(
      correctionCommandSchema.safeParse({
        replacement: command,
        transactionId: "hidden",
      }).success,
    ).toBe(false);
  });
  it.each(createKinds)("freezes %s required and forbidden fields", (kind) => {
    const trade = kind === "BUY" || kind === "SELL";
    const candidate = {
      kind,
      currency: "USD",
      effectiveAt: command.effectiveAt,
      ...(trade
        ? {
            assetId: asset.id,
            quantity: "1.000000000001",
            unitPrice: "999999999999999999999999.10",
            fee: "0",
          }
        : { amount: "1" }),
      ...(kind === "DIVIDEND" ? { assetId: asset.id } : {}),
    };
    expect(commandSchema.parse(candidate)).toEqual(candidate);
    for (const key of Object.keys(candidate).filter((key) => key !== "fee")) {
      const missing: Record<string, unknown> = { ...candidate };
      delete missing[key];
      expect(commandSchema.safeParse(missing).success).toBe(false);
    }
    const forbidden = trade
      ? ["amount"]
      : kind === "DIVIDEND"
        ? ["quantity", "unitPrice", "fee"]
        : ["assetId", "quantity", "unitPrice", "fee"];
    for (const key of [
      ...forbidden,
      "ownerUserId",
      "portfolioId",
      "gross",
      "net",
    ])
      expect(
        commandSchema.safeParse({ ...candidate, [key]: "1" }).success,
      ).toBe(false);
  });
  it.each(["REVERSAL", "ADJUSTMENT", "unknown"])(
    "rejects create kind %s",
    (kind) =>
      expect(commandSchema.safeParse({ ...command, kind }).success).toBe(false),
  );
  it.each([
    "-1",
    "+1",
    "1e2",
    "01",
    ".1",
    "1.",
    "1.0000000000001",
    " 1",
    "1 ",
    "NaN",
    1,
  ])("rejects invalid decimal %s", (value) =>
    expect(decimal.safeParse(value).success).toBe(false),
  );
  it("preserves precision and rejects zero only for positive fields", () => {
    expect(
      positiveDecimal.parse("123456789012345678901234567890.123456789012"),
    ).toBe("123456789012345678901234567890.123456789012");
    expect(decimal.parse("0.000")).toBe("0.000");
    expect(positiveDecimal.safeParse("0.000").success).toBe(false);
  });
  it.each([
    "2020-02-30T00:00:00Z",
    "2020-01-01T24:00:00Z",
    "2020-01-01T00:00:00+00:00",
    "2020-01-01T00:00:00.1234567Z",
  ])("rejects invalid UTC time %s", (value) =>
    expect(historyTime.safeParse(value).success).toBe(false),
  );
  it("allows future filter bounds but not future commands, and checks microsecond range ordering", () => {
    const future = "2999-01-01T00:00:00.123456Z";
    expect(
      historyFiltersSchema.parse({
        effectiveAtTo: future,
        includeReversals: true,
      }).effectiveAtTo,
    ).toBe(future);
    expect(
      commandSchema.safeParse({ ...command, effectiveAt: future }).success,
    ).toBe(false);
    expect(
      historyFiltersSchema.safeParse({
        effectiveAtFrom: "2020-01-01T00:00:00.000002Z",
        effectiveAtTo: "2020-01-01T00:00:00.000001Z",
        includeReversals: true,
      }).success,
    ).toBe(false);
  });
  it("preserves optional text semantics, Unicode bounds, and enforces the UTF-8 body limit", () => {
    expect(
      commandSchema.parse({
        ...command,
        note: " \u00e9 ",
        externalReference: "",
      }),
    ).toEqual({ ...command, note: " \u00e9 ", externalReference: "" });
    expect(commandSchema.parse(command)).not.toHaveProperty("note");
    expect(
      commandSchema.safeParse({ ...command, note: "x".repeat(2001) }).success,
    ).toBe(false);
    expect(
      commandSchema.safeParse({
        ...command,
        externalReference: "x".repeat(257),
      }).success,
    ).toBe(false);
    expect(
      commandSchema.safeParse({
        ...command,
        note: "😀".repeat(2000),
        externalReference: "😀".repeat(256),
      }).success,
    ).toBe(false);
    expect(
      commandSchema.safeParse({ ...command, currency: "EUR" }).success,
    ).toBe(false);
  });
  it("permits only canonical USD US Equity/ETF selection", () => {
    for (const exchange of ["NYSE", "NASDAQ", "NYSEARCA", "AMEX"])
      for (const assetType of ["EQUITY", "ETF"] as const)
        expect(isEligibleAsset({ ...asset, exchange, assetType })).toBe(true);
    expect(
      isEligibleAsset({ ...asset, assetType: "CRYPTO", exchange: "CRYPTO" }),
    ).toBe(false);
    expect(isEligibleAsset({ ...asset, exchange: "LSE" })).toBe(false);
  });
});
