import { afterEach, describe, expect, it, vi } from "vitest";
import { transactionApi } from "@/features/transaction/api/transaction-api";
import { command, transaction } from "@/features/transaction/test-fixtures";

afterEach(() => vi.unstubAllGlobals());
function respond(body: unknown, status = 200) {
  const fetch = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(body), {
      status,
      headers: { "X-Correlation-ID": "safe-ref" },
    }),
  );
  vi.stubGlobal("fetch", fetch);
  return fetch;
}

describe("Transaction API", () => {
  it("uses generated facts, Bearer and idempotency headers without cookies", async () => {
    const fetch = respond(transaction, 201);
    expect(
      await transactionApi.create(
        "memory-access",
        "p/id",
        command,
        "attempt-key-value",
      ),
    ).toEqual(transaction);
    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/portfolios/p%2Fid/transactions",
      expect.objectContaining({
        method: "POST",
        credentials: "omit",
        body: JSON.stringify(command),
        headers: {
          Accept: "application/json",
          Authorization: "Bearer memory-access",
          "Content-Type": "application/json",
          "Idempotency-Key": "attempt-key-value",
        },
      }),
    );
  });
  it("encodes filters and round-trips opaque cursors unchanged, without a command key", async () => {
    const cursor = "opaque.+/=?&value";
    const fetch = respond({ items: [transaction], nextCursor: cursor });
    expect(
      await transactionApi.list("memory-access", "p", {
        cursor,
        kind: "REVERSAL",
        effectiveAtFrom: "2999-01-01T00:00:00Z",
        includeReversals: false,
        limit: 50,
      }),
    ).toEqual({ items: [transaction], nextCursor: cursor });
    const url = new URL(fetch.mock.calls[0][0], "https://app.localhost:3443");
    expect(url.searchParams.get("cursor")).toBe(cursor);
    expect(url.searchParams.get("kind")).toBe("REVERSAL");
    expect(url.searchParams.get("includeReversals")).toBe("false");
    expect(fetch.mock.calls[0][1].headers).not.toHaveProperty(
      "Idempotency-Key",
    );
  });
  it.each([
    { ...transaction, amount: 10 },
    { ...transaction, ownerUserId: "private" },
    { ...transaction, correctionLinks: {} },
    { ...transaction, portfolioSequence: 1 },
    { ...transaction, amount: "-1" },
    { ...transaction, effectiveAt: "invalid" },
  ])("rejects malformed success", async (body) => {
    respond(body, 201);
    await expect(
      transactionApi.create("memory-access", "p", command, "attempt-key-value"),
    ).rejects.toMatchObject({
      code: "INTERNAL_ERROR",
      correlationId: "safe-ref",
    });
  });
  it("preserves public error codes/correlation without presenting raw backend messages", async () => {
    respond(
      {
        error: {
          code: "INVALID_BACKDATED_LEDGER",
          message: "unsafe internal detail",
          correlationId: "request-ref",
        },
      },
      422,
    );
    await expect(
      transactionApi.create("memory-access", "p", command, "attempt-key-value"),
    ).rejects.toMatchObject({
      status: 422,
      code: "INVALID_BACKDATED_LEDGER",
      correlationId: "request-ref",
      message: "Transaction service is unavailable. Please try again.",
    });
  });
  it("fails closed for malformed errors and wrong success status", async () => {
    respond({ error: "private detail" }, 500);
    await expect(
      transactionApi.list("memory-access", "p", {}),
    ).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
    respond(transaction, 200);
    await expect(
      transactionApi.create("memory-access", "p", command, "attempt-key-value"),
    ).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
  });
  it("collapses network failures and invalid JSON", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new Error("private network detail")),
    );
    await expect(
      transactionApi.list("memory-access", "p", {}),
    ).rejects.toMatchObject({ status: 0, code: "INTERNAL_ERROR" });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("invalid")));
    await expect(
      transactionApi.list("memory-access", "p", {}),
    ).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
  });
});
