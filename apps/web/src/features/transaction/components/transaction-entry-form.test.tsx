import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TransactionEntryForm } from "@/features/transaction/components/transaction-entry-form";
import {
  asset,
  command,
  transaction,
} from "@/features/transaction/test-fixtures";
import { ApiError } from "@/platform/api/api-error";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  reset: vi.fn(),
  assets: vi.fn(),
}));
vi.mock("@/features/transaction/model/transaction-queries", () => ({
  useCreateTransaction: () => ({
    mutateAsync: mocks.create,
    reset: mocks.reset,
  }),
}));
vi.mock("@/features/asset/model/asset-queries", () => ({
  useAssets: (...args: unknown[]) => mocks.assets(...args),
}));
beforeEach(() => {
  vi.clearAllMocks();
  mocks.create.mockResolvedValue(transaction);
  mocks.assets.mockReturnValue({
    isSuccess: true,
    data: {
      pages: [
        {
          items: [
            asset,
            {
              ...asset,
              id: "crypto",
              symbol: "CRYPTO-SYNTH",
              assetType: "CRYPTO",
              exchange: "CRYPTO",
            },
          ],
        },
      ],
    },
  });
});
const fill = (label: string | RegExp, value: string) =>
  fireEvent.change(screen.getByLabelText(label), { target: { value } });
function deposit() {
  fill("Transaction kind", "DEPOSIT");
  fill("Amount (USD)", "10.000");
  fill("Effective time (UTC)", command.effectiveAt);
}
async function review() {
  fireEvent.click(screen.getByRole("button", { name: "Review entry" }));
  await screen.findByRole("heading", {
    name: "Review entry before submitting",
  });
}

describe("reviewed Transaction entry", () => {
  it("offers six kinds, canonical eligible Assets, and no reversal/adjustment", () => {
    render(<TransactionEntryForm portfolioId="p" />);
    expect(
      within(screen.getByLabelText("Transaction kind"))
        .getAllByRole("option")
        .map((option) => option.textContent),
    ).toEqual(["BUY", "SELL", "DIVIDEND", "DEPOSIT", "WITHDRAWAL", "FEE"]);
    expect(
      screen.getByRole("button", { name: /SYNTH — Synthetic Equity/ }),
    ).toBeVisible();
    expect(screen.queryByText(/CRYPTO-SYNTH/)).not.toBeInTheDocument();
  });
  it.each(["BUY", "SELL", "DIVIDEND", "DEPOSIT", "WITHDRAWAL", "FEE"])(
    "submits reviewed %s string facts only",
    async (kind) => {
      render(<TransactionEntryForm portfolioId="p" />);
      fill("Transaction kind", kind);
      const trade = kind === "BUY" || kind === "SELL";
      if (trade || kind === "DIVIDEND")
        fireEvent.click(
          screen.getByRole("button", { name: /SYNTH — Synthetic Equity/ }),
        );
      if (trade) {
        fill("Quantity", "1.000000000001");
        fill("Unit price (USD)", "2.10");
        fill(/Fee \(USD/, "0.00");
      } else fill("Amount (USD)", "10.000");
      fill("Effective time (UTC)", command.effectiveAt);
      await review();
      expect(mocks.create).not.toHaveBeenCalled();
      expect(
        screen.getByRole("heading", { name: "Review entry before submitting" }),
      ).toHaveFocus();
      fireEvent.click(
        screen.getByRole("button", { name: "Confirm submission" }),
      );
      await screen.findByText("Transaction accepted");
      expect(mocks.create.mock.calls[0][0].command).toEqual({
        kind,
        currency: "USD",
        effectiveAt: command.effectiveAt,
        ...(trade || kind === "DIVIDEND" ? { assetId: asset.id } : {}),
        ...(trade
          ? { quantity: "1.000000000001", unitPrice: "2.10", fee: "0.00" }
          : { amount: "10.000" }),
      });
      expect(screen.getByText("Immutable record: transaction-1")).toBeVisible();
    },
  );
  it("removes stale fields when switching kind and preserves explicit optional empty text", async () => {
    render(<TransactionEntryForm portfolioId="p" />);
    fireEvent.click(
      screen.getByRole("button", { name: /SYNTH — Synthetic Equity/ }),
    );
    fill("Quantity", "10");
    fill("Unit price (USD)", "2");
    fill(/Fee \(USD/, "3");
    deposit();
    fireEvent.click(screen.getByLabelText("Include note"));
    fireEvent.click(screen.getByLabelText("Include external reference"));
    fill(/External reference \(optional/, "  ref  ");
    await review();
    fireEvent.click(screen.getByRole("button", { name: "Confirm submission" }));
    await screen.findByText("Transaction accepted");
    expect(mocks.create.mock.calls[0][0].command).toEqual({
      ...command,
      note: "",
      externalReference: "  ref  ",
    });
  });
  it("keeps draft values on review return and blocks invalid input without submitting", async () => {
    render(<TransactionEntryForm portfolioId="p" />);
    deposit();
    fill("Amount (USD)", "1e3");
    fireEvent.click(screen.getByRole("button", { name: "Review entry" }));
    await screen.findByRole("alert");
    expect(screen.getByLabelText("Amount (USD)")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(mocks.create).not.toHaveBeenCalled();
    fill("Amount (USD)", "10.000");
    await review();
    fireEvent.click(screen.getByRole("button", { name: "Return to entry" }));
    expect(screen.getByLabelText("Amount (USD)")).toHaveValue("10.000");
  });
  it("reuses retry keys, replaces them only for changed commands, clears after success, and never persists", async () => {
    const storage = vi.spyOn(Storage.prototype, "setItem");
    mocks.create.mockRejectedValueOnce(
      new ApiError(0, "INTERNAL_ERROR", "private"),
    );
    render(<TransactionEntryForm portfolioId="p" />);
    deposit();
    await review();
    fireEvent.click(screen.getByRole("button", { name: "Confirm submission" }));
    await screen.findByRole("button", { name: "Retry same entry" });
    const key = mocks.create.mock.calls[0][0].key;
    mocks.create.mockRejectedValueOnce(new Error("uncertain"));
    fireEvent.click(screen.getByRole("button", { name: "Retry same entry" }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(2));
    await screen.findByRole("button", { name: "Retry same entry" });
    expect(mocks.create.mock.calls[1][0].key).toBe(key);
    fireEvent.click(screen.getByRole("button", { name: "Return to entry" }));
    fill("Amount (USD)", "11");
    await review();
    fireEvent.click(screen.getByRole("button", { name: "Confirm submission" }));
    await screen.findByText("Transaction accepted");
    const changedKey = mocks.create.mock.calls[2][0].key;
    expect(changedKey).not.toBe(key);
    fireEvent.click(
      screen.getByRole("button", { name: "Record another entry" }),
    );
    expect(screen.getByLabelText("Transaction kind")).toHaveFocus();
    deposit();
    fill("Amount (USD)", "11");
    await review();
    fireEvent.click(screen.getByRole("button", { name: "Confirm submission" }));
    await screen.findByText("Transaction accepted");
    expect(mocks.create.mock.calls[3][0].key).not.toBe(changedKey);
    expect(storage).not.toHaveBeenCalled();
    storage.mockRestore();
  });
  it("disables confirmation and editing while a request is in flight", async () => {
    mocks.create.mockReturnValue(new Promise(() => {}));
    render(<TransactionEntryForm portfolioId="p" />);
    deposit();
    await review();
    fireEvent.click(screen.getByRole("button", { name: "Confirm submission" }));
    expect(screen.getByRole("button", { name: "Submitting…" })).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Return to entry" }),
    ).toBeDisabled();
    expect(mocks.create).toHaveBeenCalledTimes(1);
  });
});
