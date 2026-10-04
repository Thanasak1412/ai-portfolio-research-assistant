import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TransactionAssetPicker } from "@/features/transaction/components/transaction-asset-picker";
import { asset } from "@/features/transaction/test-fixtures";

const query = vi.hoisted(() => vi.fn());
vi.mock("@/features/asset/model/asset-queries", () => ({ useAssets: query }));
describe("Canonical eligible Asset search", () => {
  it("uses catalog search/type filters, paging, and canonical IDs", () => {
    const next = vi.fn();
    const select = vi.fn();
    query.mockReturnValue({
      isSuccess: true,
      data: { pages: [{ items: [asset] }] },
      hasNextPage: true,
      fetchNextPage: next,
    });
    render(<TransactionAssetPicker selected={null} onSelect={select} />);
    fireEvent.change(screen.getByLabelText("Asset type"), {
      target: { value: "ETF" },
    });
    fireEvent.change(screen.getByLabelText("Search Assets"), {
      target: { value: "synthetic" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    expect(query).toHaveBeenLastCalledWith({
      search: "synthetic",
      assetType: "ETF",
      limit: 25,
    });
    fireEvent.click(screen.getByRole("button", { name: /SYNTH — Synthetic/ }));
    expect(select).toHaveBeenCalledWith(asset);
    fireEvent.click(screen.getByRole("button", { name: "Load more Assets" }));
    expect(next).toHaveBeenCalledOnce();
    fireEvent.change(screen.getByLabelText("Search Assets"), {
      target: { value: "x".repeat(101) },
    });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    expect(screen.getByRole("alert")).toHaveTextContent("100 characters");
  });
  it("shows loading, retryable failure, and eligible-empty results", () => {
    query.mockReturnValue({ isLoading: true });
    const { rerender } = render(
      <TransactionAssetPicker selected={null} onSelect={vi.fn()} />,
    );
    expect(screen.getByRole("status")).toHaveTextContent(
      "Loading eligible Assets",
    );
    const retry = vi.fn();
    query.mockReturnValue({ isError: true, refetch: retry });
    rerender(<TransactionAssetPicker selected={null} onSelect={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "Retry Assets" }));
    expect(retry).toHaveBeenCalledOnce();
    query.mockReturnValue({
      isSuccess: true,
      data: { pages: [{ items: [{ ...asset, exchange: "LSE" }] }] },
    });
    rerender(<TransactionAssetPicker selected={null} onSelect={vi.fn()} />);
    expect(screen.getByText(/No eligible Assets/)).toBeVisible();
    expect(
      screen.queryByRole("button", { name: /SYNTH/ }),
    ).not.toBeInTheDocument();
  });
});
