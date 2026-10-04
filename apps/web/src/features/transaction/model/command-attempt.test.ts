import { describe, expect, it, vi } from "vitest";
import {
  CommandAttempt,
  CorrectionAttempt,
} from "@/features/transaction/model/command-attempt";
import { command } from "@/features/transaction/test-fixtures";

describe("memory-only command attempt", () => {
  it("isolates correction scope and target, canonicalizes semantic retries, and clears on success without persistence", () => {
    const storage = vi.spyOn(Storage.prototype, "setItem");
    const cookie = vi.spyOn(document, "cookie", "set");
    const attempt = new CorrectionAttempt();
    const first = attempt.keyFor("original", command);
    expect(first).toMatch(/^[A-Za-z0-9][A-Za-z0-9._~-]{15,127}$/);
    expect(attempt.keyFor("original", { ...command, amount: "10" })).toBe(
      first,
    );
    const other = attempt.keyFor("other-target", command);
    expect(other).not.toBe(first);
    const edit = attempt.keyFor("other-target", { ...command, note: "" });
    expect(edit).not.toBe(other);
    expect(attempt.keyFor("other-target", { ...command, note: " " })).not.toBe(
      edit,
    );
    const trade = {
      kind: "BUY",
      currency: "USD",
      assetId: "a",
      quantity: "1.00",
      unitPrice: "2.00",
      effectiveAt: "2020-01-01T00:00:00Z",
    } as const;
    const tradeKey = attempt.keyFor("original", trade);
    expect(
      attempt.keyFor("original", {
        ...trade,
        quantity: "1",
        fee: "0.00",
        effectiveAt: "2020-01-01T00:00:00.000000Z",
      }),
    ).toBe(tradeKey);
    attempt.clear();
    expect(attempt.keyFor("original", trade)).not.toBe(tradeKey);
    expect(new CommandAttempt().keyFor(command)).not.toBe(first);
    expect(storage).not.toHaveBeenCalled();
    expect(cookie).not.toHaveBeenCalled();
    storage.mockRestore();
    cookie.mockRestore();
  });
  it("reuses a semantic retry, changes keys for edited commands, discards on success", () => {
    const storage = vi.spyOn(Storage.prototype, "setItem");
    const attempt = new CommandAttempt();
    const key = attempt.keyFor(command);
    expect(key).toMatch(/^[A-Za-z0-9][A-Za-z0-9._~-]{15,127}$/);
    expect(attempt.keyFor({ ...command })).toBe(key);
    expect(attempt.keyFor({ ...command, amount: "10" })).toBe(key);
    const changed = attempt.keyFor({ ...command, note: "" });
    expect(changed).not.toBe(key);
    expect(attempt.keyFor({ ...command, note: " " })).not.toBe(changed);
    const beforeSuccess = attempt.keyFor(command);
    attempt.clear();
    expect(attempt.keyFor(command)).not.toBe(beforeSuccess);
    expect(new CommandAttempt().keyFor(command)).not.toBe(key);
    expect(storage).not.toHaveBeenCalled();
    storage.mockRestore();
  });
  it("treats omitted trade fee and zero as the same semantic attempt", () => {
    const attempt = new CommandAttempt();
    const trade = {
      kind: "BUY",
      assetId: "a",
      currency: "USD",
      effectiveAt: "2020-01-01T00:00:00Z",
      quantity: "1.00",
      unitPrice: "2.00",
    } as const;
    expect(
      attempt.keyFor({
        ...trade,
        fee: "0.000",
        effectiveAt: "2020-01-01T00:00:00.000000Z",
      }),
    ).toBe(attempt.keyFor(trade));
  });
});
