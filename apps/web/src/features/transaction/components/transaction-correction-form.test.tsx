import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TransactionCorrectionForm } from "@/features/transaction/components/transaction-correction-form";
import {
  asset,
  command,
  correction,
  transaction,
} from "@/features/transaction/test-fixtures";
import { transactionKeys } from "@/features/transaction/model/transaction-query-keys";
import { ApiError } from "@/platform/api/api-error";

const mocks = vi.hoisted(() => ({
  correct: vi.fn(),
  reset: vi.fn(),
  assets: vi.fn(),
}));
vi.mock("@/features/transaction/model/transaction-queries", () => ({
  useCorrectTransaction: () => ({
    mutateAsync: mocks.correct,
    reset: mocks.reset,
  }),
}));
vi.mock("@/features/asset/model/asset-queries", () => ({
  useAssets: () => mocks.assets(),
}));
beforeEach(() => {
  vi.resetAllMocks();
  mocks.correct.mockResolvedValue(correction);
  mocks.assets.mockReturnValue({
    isSuccess: true,
    data: { pages: [{ items: [asset] }] },
  });
});
function setup() {
  const client = new QueryClient();
  const close = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <TransactionCorrectionForm
        portfolioId="p"
        original={transaction}
        onClose={close}
      />
      <h2 id="history-title" tabIndex={-1}>
        Ledger history
      </h2>
    </QueryClientProvider>,
  );
  return { client, close };
}
const fill = (label: string | RegExp, value: string) =>
  fireEvent.change(screen.getByLabelText(label), { target: { value } });
function deposit() {
  fill("Replacement kind", "DEPOSIT");
  fill("Amount (USD)", command.amount);
  fill("Effective time (UTC)", command.effectiveAt);
}
async function review() {
  fireEvent.click(screen.getByRole("button", { name: "Review correction" }));
  await screen.findByRole("heading", { name: "Review correction" });
}
const confirm = () =>
  fireEvent.click(screen.getByRole("button", { name: "Confirm correction" }));

describe("immutable correction workflow", () => {
  it.each(["BUY", "SELL", "DIVIDEND", "DEPOSIT", "WITHDRAWAL", "FEE"])(
    "reviews complete %s replacement without altering original facts",
    async (kind) => {
      const { close } = setup();
      expect(
        screen.getByRole("heading", { name: "Correct Transaction" }),
      ).toHaveFocus();
      const original = screen.getByRole("region", {
        name: "Original — remains in history",
      });
      expect(within(original).getByText(transaction.id)).toBeVisible();
      expect(within(original).queryByRole("textbox")).not.toBeInTheDocument();
      expect(
        within(screen.getByLabelText("Replacement kind"))
          .getAllByRole("option")
          .map((option) => option.textContent),
      ).toEqual(["BUY", "SELL", "DIVIDEND", "DEPOSIT", "WITHDRAWAL", "FEE"]);
      fill("Replacement kind", kind);
      const trade = kind === "BUY" || kind === "SELL";
      if (trade || kind === "DIVIDEND")
        fireEvent.click(screen.getByRole("button", { name: /SYNTH —/ }));
      if (trade) {
        fill("Quantity", "1.000000000001");
        fill("Unit price (USD)", "2.10");
        fill(/Fee \(USD/, "0.00");
      } else fill("Amount (USD)", command.amount);
      fill("Effective time (UTC)", command.effectiveAt);
      fireEvent.click(screen.getByLabelText("Include note"));
      fireEvent.click(screen.getByLabelText("Include external reference"));
      fill(/External reference \(optional/, "  ref  ");
      await review();
      expect(mocks.correct).not.toHaveBeenCalled();
      expect(
        screen.getByRole("heading", { name: "Review correction" }),
      ).toHaveFocus();
      expect(
        screen.getByRole("region", {
          name: "Replacement — new facts to be recorded",
        }),
      ).toBeVisible();
      expect(screen.getByText(/will not be edited or deleted/)).toBeVisible();
      confirm();
      await screen.findByRole("heading", { name: "Correction accepted" });
      expect(
        screen.getByRole("heading", { name: "Correction accepted" }),
      ).toHaveFocus();
      expect(mocks.correct.mock.calls[0][0]).toEqual({
        transactionId: transaction.id,
        key: expect.any(String),
        command: {
          replacement: {
            kind,
            currency: "USD",
            effectiveAt: command.effectiveAt,
            note: "",
            externalReference: "  ref  ",
            ...(trade || kind === "DIVIDEND" ? { assetId: asset.id } : {}),
            ...(trade
              ? { quantity: "1.000000000001", unitPrice: "2.10", fee: "0.00" }
              : { amount: command.amount }),
          },
        },
      });
      for (const role of ["original", "reversal", "replacement"] as const)
        expect(
          within(screen.getByRole("region", { name: role })).getByText(
            correction[role].id,
          ),
        ).toBeVisible();
      fireEvent.click(
        screen.getByRole("button", { name: "Return to history" }),
      );
      expect(close).toHaveBeenCalledOnce();
    },
  );
  it("validates replacement fields accessibly and keeps exact draft on review return", async () => {
    setup();
    deposit();
    fill("Amount (USD)", "1e3");
    fireEvent.click(screen.getByRole("button", { name: "Review correction" }));
    await screen.findByRole("alert");
    expect(screen.getByLabelText("Amount (USD)")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(screen.getByLabelText("Amount (USD)")).toHaveAttribute(
      "aria-describedby",
      "replacement-amount-error",
    );
    expect(mocks.correct).not.toHaveBeenCalled();
    fill("Amount (USD)", "10.000");
    await review();
    fireEvent.click(
      screen.getByRole("button", { name: "Return to replacement" }),
    );
    expect(screen.getByLabelText("Amount (USD)")).toHaveValue("10.000");
    expect(screen.getByLabelText("Replacement kind")).toHaveFocus();
  });
  it("keeps uncertain retries stable through history inspection, but changed facts receive a new memory-only key", async () => {
    const storage = vi.spyOn(Storage.prototype, "setItem");
    const { client } = setup();
    const invalidate = vi
      .spyOn(client, "invalidateQueries")
      .mockResolvedValue();
    mocks.correct.mockRejectedValueOnce(
      new ApiError(0, "INTERNAL_ERROR", "private"),
    );
    deposit();
    await review();
    confirm();
    await screen.findByRole("button", { name: "Retry same correction" });
    const first = mocks.correct.mock.calls[0][0];
    expect(screen.getByText(/Acceptance may be uncertain/)).toBeVisible();
    fireEvent.click(
      screen.getByRole("button", { name: "Refresh ledger history" }),
    );
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: transactionKeys.histories("p"),
    });
    expect(
      screen.getByRole("heading", { name: "Ledger history" }),
    ).toHaveFocus();
    mocks.correct.mockRejectedValueOnce(
      new ApiError(503, "INTERNAL_ERROR", "private"),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Retry same correction" }),
    );
    await waitFor(() => expect(mocks.correct).toHaveBeenCalledTimes(2));
    await screen.findByRole("button", { name: "Retry same correction" });
    expect(mocks.correct.mock.calls[1][0]).toEqual(first);
    fireEvent.click(
      screen.getByRole("button", { name: "Return to replacement" }),
    );
    fill("Amount (USD)", "11");
    await review();
    confirm();
    await screen.findByText("Correction accepted");
    expect(mocks.correct.mock.calls[2][0].key).not.toBe(first.key);
    expect(storage).not.toHaveBeenCalled();
    storage.mockRestore();
  });
  it.each([
    "TRANSACTION_ALREADY_CORRECTED",
    "TRANSACTION_NOT_CORRECTABLE",
    "PORTFOLIO_ARCHIVED",
    "TRANSACTION_NOT_FOUND",
  ])("stops re-submission on %s and offers history recovery", async (code) => {
    setup();
    mocks.correct.mockRejectedValue(
      new ApiError(409, code, "private", "safe-ref"),
    );
    deposit();
    await review();
    confirm();
    await screen.findByRole("alert");
    expect(screen.queryByText("private")).not.toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Support reference: safe-ref",
    );
    expect(
      screen.queryByRole("button", {
        name: /Confirm correction|Retry same correction|Return to replacement/,
      }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Refresh ledger history" }),
    ).toBeEnabled();
    if (code === "TRANSACTION_ALREADY_CORRECTED")
      expect(screen.getByRole("alert")).toHaveTextContent(
        "select the replacement Transaction",
      );
    if (code === "TRANSACTION_NOT_CORRECTABLE")
      expect(screen.getByRole("alert")).toHaveTextContent(
        "Internal reversals cannot be corrected",
      );
    expect(mocks.correct).toHaveBeenCalledOnce();
  });
  it.each([
    "IDEMPOTENCY_CONFLICT",
    "INVALID_BACKDATED_LEDGER",
    "INSUFFICIENT_ORDERED_ASSET_QUANTITY",
    "ASSET_NOT_FOUND",
    "ASSET_FINANCIALLY_INELIGIBLE",
    "INVALID_EFFECTIVE_AT",
    "INVALID_TRANSACTION_FIELDS",
    "INVALID_DECIMAL",
  ])(
    "allows corrected facts after deterministic %s without automatic resubmission",
    async (code) => {
      setup();
      mocks.correct.mockRejectedValueOnce(new ApiError(422, code, "private"));
      deposit();
      await review();
      confirm();
      await screen.findByRole("alert");
      expect(
        screen.queryByText(/Acceptance may be uncertain/),
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Retry same correction" }),
      ).not.toBeInTheDocument();
      const key = mocks.correct.mock.calls[0][0].key;
      fireEvent.click(
        screen.getByRole("button", { name: "Return to replacement" }),
      );
      fill("Amount (USD)", "12");
      await review();
      confirm();
      await screen.findByText("Correction accepted");
      expect(mocks.correct.mock.calls[1][0].key).not.toBe(key);
    },
  );
  it("disables confirm, edit, exit and history refresh while in flight", async () => {
    setup();
    let finish!: (value: typeof correction) => void;
    mocks.correct.mockReturnValue(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    deposit();
    await review();
    confirm();
    for (const name of [
      "Submitting correction…",
      "Return to replacement",
      "Refresh ledger history",
      "Return to history",
    ])
      expect(screen.getByRole("button", { name })).toBeDisabled();
    expect(mocks.correct).toHaveBeenCalledOnce();
    await act(async () => finish(correction));
    expect(screen.getByText("Correction accepted")).toBeVisible();
  });
});
