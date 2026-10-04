import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TransactionScreen } from "@/features/transaction/components/transaction-screen";
import { portfolio } from "@/features/transaction/test-fixtures";
import { ApiError } from "@/platform/api/api-error";

const query = vi.hoisted(() => vi.fn());
vi.mock("@/features/portfolio/model/portfolio-queries", () => ({
  usePortfolio: query,
}));
vi.mock("@/features/transaction/components/transaction-entry-form", () => ({
  TransactionEntryForm: () => <div>Entry form</div>,
}));
vi.mock("@/features/transaction/components/transaction-history", () => ({
  TransactionHistory: () => <div>History</div>,
}));
beforeEach(() => query.mockReset());

describe("Transaction route screen", () => {
  it("makes archived history read-only but permits active entry", () => {
    query.mockReturnValue({ data: portfolio });
    const { rerender } = render(
      <TransactionScreen portfolioId={portfolio.id} />,
    );
    expect(screen.getByText("Entry form")).toBeVisible();
    query.mockReturnValue({ data: { ...portfolio, status: "ARCHIVED" } });
    rerender(<TransactionScreen portfolioId={portfolio.id} />);
    expect(screen.queryByText("Entry form")).not.toBeInTheDocument();
    expect(screen.getByText("History")).toBeVisible();
    expect(screen.getByText(/archived. History is read-only/)).toBeVisible();
    expect(
      screen.getByRole("link", { name: "Back to Portfolio" }),
    ).toHaveAttribute("href", `/app/portfolios/${portfolio.id}`);
  });
  it("guards entry/history until ownership lookup succeeds and supports retry", () => {
    query.mockReturnValue({ isLoading: true });
    const { rerender } = render(<TransactionScreen portfolioId="p" />);
    expect(screen.getByRole("status")).toHaveTextContent("Loading portfolio");
    expect(screen.queryByText("Entry form")).not.toBeInTheDocument();
    const refetch = vi.fn();
    query.mockReturnValue({
      isError: true,
      error: new ApiError(404, "PORTFOLIO_NOT_FOUND", "private"),
      refetch,
    });
    rerender(<TransactionScreen portfolioId="p" />);
    expect(screen.getByRole("alert")).toHaveTextContent("Portfolio not found.");
    expect(screen.queryByText("History")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry Portfolio" }));
    expect(refetch).toHaveBeenCalledOnce();
  });
});
