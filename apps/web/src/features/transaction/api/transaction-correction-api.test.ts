import { afterEach, describe, expect, it, vi } from "vitest";
import { transactionApi } from "@/features/transaction/api/transaction-api";
import {
  command,
  correction,
  transaction,
} from "@/features/transaction/test-fixtures";

afterEach(() => vi.unstubAllGlobals());
function respond(body: unknown, status = 201) {
  const fetch = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(body), {
      status,
      headers: { "X-Correlation-ID": "safe-ref" },
    }),
  );
  vi.stubGlobal("fetch", fetch);
  return fetch;
}
const send = () =>
  transactionApi.correct(
    "memory-access",
    "p",
    transaction.id,
    { replacement: command },
    "correction-attempt-key",
  );

describe("Correction API boundary", () => {
  it("encodes both route IDs, sends the exact wrapper and headers with no cookies", async () => {
    const body = {
      ...correction,
      original: { ...correction.original, id: "t/id" },
      reversal: {
        ...correction.reversal,
        correctionLinks: {
          ...correction.reversal.correctionLinks,
          reversesTransactionId: "t/id",
        },
      },
      replacement: {
        ...correction.replacement,
        correctionLinks: {
          ...correction.replacement.correctionLinks,
          replacesTransactionId: "t/id",
        },
      },
    };
    const fetch = respond(body);
    await expect(
      transactionApi.correct(
        "memory-access",
        "p/id",
        "t/id",
        { replacement: command },
        "correction-attempt-key",
      ),
    ).resolves.toEqual(body);
    expect(fetch).toHaveBeenCalledExactlyOnceWith(
      "/api/v1/portfolios/p%2Fid/transactions/t%2Fid/corrections",
      {
        method: "POST",
        credentials: "omit",
        body: JSON.stringify({ replacement: command }),
        headers: {
          Accept: "application/json",
          Authorization: "Bearer memory-access",
          "Content-Type": "application/json",
          "Idempotency-Key": "correction-attempt-key",
        },
      },
    );
  });
  it.each([200, 202])("requires 201, not %s", async (status) => {
    respond(correction, status);
    await expect(send()).rejects.toMatchObject({
      code: "INTERNAL_ERROR",
      correlationId: "safe-ref",
    });
  });
  it.each(["original", "reversal", "replacement"] as const)(
    "strictly validates complete %s facts",
    async (role) => {
      for (const malformed of [
        undefined,
        { ...correction[role], amount: 20 },
        { ...correction[role], ownerUserId: "private" },
        { ...correction[role], correctionLinks: {} },
      ]) {
        respond({ ...correction, [role]: malformed });
        await expect(send()).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
      }
    },
  );
  it.each([
    ["original", "reversalTransactionId"],
    ["original", "replacementTransactionId"],
    ["reversal", "reversesTransactionId"],
    ["replacement", "replacesTransactionId"],
  ] as const)("rejects a broken %s.%s relationship", async (role, link) => {
    respond({
      ...correction,
      [role]: {
        ...correction[role],
        correctionLinks: {
          ...correction[role].correctionLinks,
          [link]: "wrong-id",
        },
      },
    });
    await expect(send()).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
  });
  it("rejects swapped kinds, duplicate IDs, unexpected fields, and another target", async () => {
    for (const body of [
      { ...correction, reversal: { ...correction.reversal, kind: "BUY" } },
      { ...correction, original: { ...correction.original, kind: "REVERSAL" } },
      {
        ...correction,
        replacement: { ...correction.replacement, kind: "REVERSAL" },
      },
      {
        ...correction,
        replacement: { ...correction.replacement, id: correction.reversal.id },
      },
      { ...correction, metadata: {} },
    ]) {
      respond(body);
      await expect(send()).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
    }
    respond(correction);
    await expect(
      transactionApi.correct(
        "memory-access",
        "p",
        "other",
        { replacement: command },
        "correction-attempt-key",
      ),
    ).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
  });
  it.each([
    "TRANSACTION_ALREADY_CORRECTED",
    "TRANSACTION_NOT_CORRECTABLE",
    "TRANSACTION_NOT_FOUND",
    "IDEMPOTENCY_CONFLICT",
  ])("preserves safe %s and correlation, never raw messages", async (code) => {
    respond(
      { error: { code, message: "private details", correlationId: "ref-1" } },
      409,
    );
    await expect(send()).rejects.toMatchObject({
      code,
      correlationId: "ref-1",
      message: "Transaction service is unavailable. Please try again.",
    });
  });
  it("fails closed for malformed errors, invalid JSON and network failure", async () => {
    respond({ error: "private details" }, 500);
    await expect(send()).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("bad json", { status: 201 })),
    );
    await expect(send()).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("private")));
    await expect(send()).rejects.toMatchObject({
      status: 0,
      code: "INTERNAL_ERROR",
    });
  });
  it("retrieves related records through the same owner-scoped API without a command key", async () => {
    const fetch = respond({ ...transaction, id: "t/id" }, 200);
    await transactionApi.get("memory-access", "p/id", "t/id");
    expect(fetch).toHaveBeenCalledExactlyOnceWith(
      "/api/v1/portfolios/p%2Fid/transactions/t%2Fid",
      {
        credentials: "omit",
        headers: {
          Accept: "application/json",
          Authorization: "Bearer memory-access",
        },
      },
    );
    respond(transaction, 200);
    await expect(
      transactionApi.get("memory-access", "p", "other"),
    ).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
  });
});
