import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TransactionHistory } from "@/features/transaction/components/transaction-history";
import { transaction } from "@/features/transaction/test-fixtures";

const history = vi.hoisted(() => vi.fn());
vi.mock("@/features/transaction/model/transaction-queries", () => ({
  useTransactionHistory: history,
}));
const state = (overrides = {}) => ({
  isLoading: false,
  isSuccess: true,
  data: { pages: [{ items: [transaction] }] },
  ...overrides,
});
beforeEach(() => {
  history.mockReset();
  history.mockReturnValue(state());
});

describe("Ledger history", () => {
  it("displays immutable facts in server order without numeric coercion or mutations", () => {
    history.mockReturnValue(
      state({
        data: {
          pages: [
            {
              items: [
                transaction,
                { ...transaction, id: "second", kind: "REVERSAL" },
              ],
            },
          ],
        },
      }),
    );
    render(<TransactionHistory portfolioId="p" />);
    expect(
      screen
        .getAllByRole("heading", { level: 3 })
        .map((heading) => heading.textContent),
    ).toEqual(["DEPOSIT · transaction-1", "REVERSAL · second"]);
    expect(screen.getAllByText(transaction.portfolioSequence)).toHaveLength(2);
    expect(
      screen.queryByRole("button", { name: /edit|delete|correct/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/gross|net proceeds|valuation/i),
    ).not.toBeInTheDocument();
  });
  it("supports loading, empty, retry and next-page failure with existing rows retained", () => {
    history.mockReturnValue(
      state({ isLoading: true, isSuccess: false, data: undefined }),
    );
    const { rerender } = render(<TransactionHistory portfolioId="p" />);
    expect(screen.getByRole("status")).toHaveTextContent(
      "Loading transactions",
    );
    history.mockReturnValue(state({ data: { pages: [{ items: [] }] } }));
    rerender(<TransactionHistory portfolioId="p" />);
    expect(
      screen.getByText("No transactions match these filters."),
    ).toBeVisible();
    const refetch = vi.fn();
    const fetchNextPage = vi.fn();
    history.mockReturnValue(
      state({
        isSuccess: false,
        isError: true,
        error: new Error("private"),
        refetch,
      }),
    );
    rerender(<TransactionHistory portfolioId="p" />);
    fireEvent.click(screen.getByRole("button", { name: "Retry history" }));
    expect(refetch).toHaveBeenCalledOnce();
    expect(screen.queryByText("private")).not.toBeInTheDocument();
    history.mockReturnValue(
      state({
        isSuccess: false,
        isError: true,
        isFetchNextPageError: true,
        fetchNextPage,
        hasNextPage: true,
      }),
    );
    rerender(<TransactionHistory portfolioId="p" />);
    expect(
      screen.getByRole("heading", { name: "DEPOSIT · transaction-1" }),
    ).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Retry history" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Load more transactions" }),
    );
    expect(fetchNextPage).toHaveBeenCalledTimes(2);
  });
  it("applies approved filters including future read bounds and resets them", () => {
    render(<TransactionHistory portfolioId="p" />);
    fireEvent.change(screen.getByLabelText("Kind"), {
      target: { value: "REVERSAL" },
    });
    fireEvent.change(screen.getByLabelText("Effective from (UTC, inclusive)"), {
      target: { value: "2999-01-01T00:00:00Z" },
    });
    fireEvent.change(screen.getByLabelText("Effective to (UTC, inclusive)"), {
      target: { value: "2999-12-31T00:00:00Z" },
    });
    fireEvent.click(screen.getByLabelText("Include internal reversals"));
    fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));
    expect(history).toHaveBeenLastCalledWith("p", {
      kind: "REVERSAL",
      effectiveAtFrom: "2999-01-01T00:00:00Z",
      effectiveAtTo: "2999-12-31T00:00:00Z",
      includeReversals: false,
    });
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(history).toHaveBeenLastCalledWith("p", { includeReversals: true });
    fireEvent.change(screen.getByLabelText("Effective from (UTC, inclusive)"), {
      target: { value: "invalid" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));
    expect(screen.getByRole("alert")).toBeVisible();
    expect(history).toHaveBeenLastCalledWith("p", { includeReversals: true });
  });
});
