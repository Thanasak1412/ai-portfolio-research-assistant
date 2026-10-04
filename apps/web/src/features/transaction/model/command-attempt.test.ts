import { describe, expect, it, vi } from "vitest";
import { CommandAttempt } from "@/features/transaction/model/command-attempt";
import { command } from "@/features/transaction/test-fixtures";

describe("memory-only command attempt", () => {
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
