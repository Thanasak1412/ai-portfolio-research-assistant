import { z } from "zod";

import type { Asset } from "@/features/asset/api/asset-api";

export const createKinds = [
  "BUY",
  "SELL",
  "DIVIDEND",
  "DEPOSIT",
  "WITHDRAWAL",
  "FEE",
] as const;
export const visibleKinds = [...createKinds, "REVERSAL"] as const;
export const decimal = z
  .string()
  .regex(
    /^(?:0|[1-9][0-9]*)(?:\.[0-9]{1,12})?$/,
    "Use a decimal string with at most 12 fractional digits.",
  );
export const positiveDecimal = decimal.refine(
  (value) => /[1-9]/.test(value),
  "Must be greater than zero.",
);

// Compare fixed-width UTC strings to retain all six fractional digits. Date is
// used only to reject impossible calendar dates, never for financial values.
export function canonicalTime(value: string): string {
  const [seconds, fraction = ""] = value.slice(0, -1).split(".");
  return `${seconds}.${fraction.padEnd(6, "0")}Z`;
}
export const historyTime = z
  .string()
  .regex(
    /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/,
    "Use UTC time: YYYY-MM-DDTHH:mm:ss[.ffffff]Z.",
  )
  .refine((value) => {
    const date = new Date(value);
    return (
      Number.isFinite(date.getTime()) &&
      date.toISOString().slice(0, 19) === value.slice(0, 19)
    );
  }, "Enter a valid UTC calendar date and time.");
export const commandTime = historyTime.refine(
  (value) => canonicalTime(value) <= canonicalTime(new Date().toISOString()),
  "Effective time cannot be in the future.",
);
const boundedText = (limit: number) =>
  z
    .string()
    .refine(
      (value) => Array.from(value).length <= limit,
      `Use at most ${limit} characters.`,
    );
export const opaqueId = z.string().min(1).max(128);
const common = {
  currency: z.literal("USD"),
  effectiveAt: commandTime,
  note: boundedText(2000).optional(),
  externalReference: boundedText(256).optional(),
};
const trade = {
  ...common,
  assetId: opaqueId,
  quantity: positiveDecimal,
  unitPrice: positiveDecimal,
  fee: decimal.optional(),
};
export const commandSchema = z
  .discriminatedUnion("kind", [
    z.strictObject({ ...trade, kind: z.literal("BUY") }),
    z.strictObject({ ...trade, kind: z.literal("SELL") }),
    z.strictObject({
      ...common,
      kind: z.literal("DIVIDEND"),
      assetId: opaqueId,
      amount: positiveDecimal,
    }),
    z.strictObject({
      ...common,
      kind: z.literal("DEPOSIT"),
      amount: positiveDecimal,
    }),
    z.strictObject({
      ...common,
      kind: z.literal("WITHDRAWAL"),
      amount: positiveDecimal,
    }),
    z.strictObject({
      ...common,
      kind: z.literal("FEE"),
      amount: positiveDecimal,
    }),
  ])
  .refine(
    (command) =>
      new TextEncoder().encode(JSON.stringify(command)).length <= 8192,
    "The entry must fit within 8,192 UTF-8 bytes.",
  );

export function isEligibleAsset(asset: Asset): boolean {
  return (
    (asset.assetType === "EQUITY" || asset.assetType === "ETF") &&
    asset.currency === "USD" &&
    ["NYSE", "NASDAQ", "NYSEARCA", "AMEX"].includes(asset.exchange)
  );
}

export const historyFiltersSchema = z
  .object({
    kind: z.enum(visibleKinds).optional(),
    effectiveAtFrom: historyTime.optional(),
    effectiveAtTo: historyTime.optional(),
    includeReversals: z.boolean(),
  })
  .refine(
    ({ effectiveAtFrom, effectiveAtTo }) =>
      !effectiveAtFrom ||
      !effectiveAtTo ||
      canonicalTime(effectiveAtFrom) <= canonicalTime(effectiveAtTo),
    {
      message: "The end time must not precede the start time.",
      path: ["effectiveAtTo"],
    },
  );

// Success validation deliberately does not apply the command's client-clock check.
export const transactionSchema = z.strictObject({
  id: opaqueId,
  kind: z.enum(visibleKinds),
  assetId: opaqueId.nullable(),
  quantity: positiveDecimal.nullable(),
  unitPrice: positiveDecimal.nullable(),
  fee: decimal.nullable(),
  amount: positiveDecimal.nullable(),
  currency: z.literal("USD"),
  effectiveAt: historyTime,
  portfolioSequence: z.string().regex(/^[1-9][0-9]*$/),
  note: boundedText(2000).nullable(),
  externalReference: boundedText(256).nullable(),
  correctionLinks: z.strictObject({
    reversesTransactionId: opaqueId.nullable(),
    replacesTransactionId: opaqueId.nullable(),
    reversalTransactionId: opaqueId.nullable(),
    replacementTransactionId: opaqueId.nullable(),
  }),
  createdAt: z.iso.datetime(),
});
