import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TransactionHistory } from "@/features/transaction/components/transaction-history";
import { transactionAnchor } from "@/features/transaction/components/transaction-record";
import { correction, transaction } from "@/features/transaction/test-fixtures";
import { ApiError } from "@/platform/api/api-error";

const mocks = vi.hoisted(() => ({ history: vi.fn(), detail: vi.fn() }));
vi.mock("@/features/transaction/model/transaction-queries", () => ({
  useTransactionHistory: mocks.history,
  useTransaction: mocks.detail,
}));
beforeEach(() => {
  vi.resetAllMocks();
  mocks.history.mockReturnValue({
    isSuccess: true,
    data: {
      pages: [
        {
          items: [
            correction.original,
            correction.reversal,
            correction.replacement,
            { ...transaction, id: "uncorrected" },
          ],
        },
      ],
    },
  });
});
const row = (id: string) => document.getElementById(transactionAnchor(id))!;

describe("immutable history relationships", () => {
  it("offers Correct for an uncorrected public row and replacement, never for reversal or corrected original", () => {
    const correct = vi.fn();
    const { rerender } = render(
      <TransactionHistory portfolioId="p" onCorrect={correct} />,
    );
    for (const record of [correction.original, correction.reversal])
      expect(
        within(row(record.id)).queryByRole("button", { name: "Correct" }),
      ).not.toBeInTheDocument();
    for (const id of [correction.replacement.id, "uncorrected"]) {
      fireEvent.click(within(row(id)).getByRole("button", { name: "Correct" }));
      expect(correct).toHaveBeenLastCalledWith(expect.objectContaining({ id }));
    }
    expect(
      screen.queryByRole("button", { name: /edit|delete/i }),
    ).not.toBeInTheDocument();
    rerender(
      <TransactionHistory
        portfolioId="p"
        onCorrect={correct}
        correctionPending
      />,
    );
    for (const button of screen.getAllByRole("button", { name: "Correct" }))
      expect(button).toBeDisabled();
    // No mutation capability is passed for archived Portfolios; links remain.
    rerender(<TransactionHistory portfolioId="p" />);
    expect(
      screen.queryByRole("button", { name: "Correct" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("link", {
        name: `Replacement: ${correction.replacement.id}`,
      }),
    ).toBeVisible();
  });
  it("shows both generations of a chain and focuses the actual linked immutable record", () => {
    const intermediate = {
      ...correction.replacement,
      correctionLinks: {
        ...correction.replacement.correctionLinks,
        reversalTransactionId: "R2",
        replacementTransactionId: "C",
      },
    };
    const secondReversal = {
      ...correction.reversal,
      id: "R2",
      correctionLinks: {
        ...transaction.correctionLinks,
        reversesTransactionId: intermediate.id,
      },
    };
    const lastReplacement = {
      ...transaction,
      id: "C",
      correctionLinks: {
        ...transaction.correctionLinks,
        replacesTransactionId: intermediate.id,
      },
    };
    mocks.history.mockReturnValue({
      isSuccess: true,
      data: {
        pages: [
          { items: [correction.original, correction.reversal] },
          { items: [intermediate, secondReversal, lastReplacement] },
        ],
      },
    });
    render(<TransactionHistory portfolioId="p" onCorrect={vi.fn()} />);
    const relations = [
      [correction.original.id, "Reversal", correction.reversal.id],
      [correction.original.id, "Replacement", intermediate.id],
      [correction.reversal.id, "Reverses", correction.original.id],
      [intermediate.id, "Replaces", correction.original.id],
      [intermediate.id, "Reversal", "R2"],
      [intermediate.id, "Replacement", "C"],
      ["R2", "Reverses", intermediate.id],
      ["C", "Replaces", intermediate.id],
    ];
    for (const [source, label, target] of relations) {
      const link = within(row(source)).getByRole("link", {
        name: `${label}: ${target}`,
      });
      expect(link).toHaveAttribute("href", `#${transactionAnchor(target)}`);
      fireEvent.click(link);
      expect(row(target)).toHaveFocus();
    }
    expect(
      within(row(intermediate.id)).queryByRole("button", { name: "Correct" }),
    ).not.toBeInTheDocument();
    expect(
      within(row("C")).getByRole("button", { name: "Correct" }),
    ).toBeEnabled();
    expect(mocks.detail).not.toHaveBeenCalled();
  });
  it("loads a relationship outside current filters/pages with safe failure/retry and full chain navigation", () => {
    mocks.history.mockReturnValue({
      isSuccess: true,
      data: { pages: [{ items: [correction.original] }] },
    });
    mocks.detail.mockReturnValue({ isLoading: true });
    const correct = vi.fn();
    const { rerender } = render(
      <TransactionHistory portfolioId="p" onCorrect={correct} />,
    );
    fireEvent.click(
      screen.getByRole("button", {
        name: `Replacement: ${correction.replacement.id}`,
      }),
    );
    expect(mocks.detail).toHaveBeenLastCalledWith(
      "p",
      correction.replacement.id,
    );
    expect(
      screen.getByRole("heading", { name: "Related Transaction" }),
    ).toHaveFocus();
    expect(screen.getByRole("status")).toHaveTextContent(
      "Loading related Transaction",
    );
    const retry = vi.fn();
    mocks.detail.mockReturnValue({
      isError: true,
      error: new ApiError(404, "TRANSACTION_NOT_FOUND", "private"),
      refetch: retry,
    });
    rerender(<TransactionHistory portfolioId="p" onCorrect={correct} />);
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Transaction not found",
    );
    expect(screen.queryByText("private")).not.toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Retry related Transaction" }),
    );
    expect(retry).toHaveBeenCalledOnce();
    mocks.detail.mockReturnValue({ data: correction.replacement });
    rerender(<TransactionHistory portfolioId="p" onCorrect={correct} />);
    const related = screen.getByRole("region", { name: "Related Transaction" });
    expect(
      within(related).getByRole("heading", {
        name: `DEPOSIT · ${correction.replacement.id}`,
      }),
    ).toBeVisible();
    fireEvent.click(within(related).getByRole("button", { name: "Correct" }));
    expect(correct).toHaveBeenCalledWith(correction.replacement);
    fireEvent.click(
      within(related).getByRole("link", {
        name: `Replaces: ${transaction.id}`,
      }),
    );
    expect(row(transaction.id)).toHaveFocus();
    fireEvent.click(
      screen.getByRole("button", { name: "Close related Transaction" }),
    );
    expect(
      screen.queryByRole("region", { name: "Related Transaction" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Ledger history" }),
    ).toHaveFocus();
  });
});
