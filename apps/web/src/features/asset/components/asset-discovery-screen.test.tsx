import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AssetApiError } from "@/features/asset/api/asset-api";
import { AssetDiscoveryScreen } from "@/features/asset/components/asset-discovery-screen";

const useAssets = vi.fn();

vi.mock("@/features/asset/model/asset-queries", async (importOriginal) => {
  const original =
    await importOriginal<
      typeof import("@/features/asset/model/asset-queries")
    >();
  return { ...original, useAssets: (...args: unknown[]) => useAssets(...args) };
});

const equity = {
  id: "asset-equity",
  symbol: "ACME",
  name: "Acme Corporation",
  assetType: "EQUITY" as const,
  exchange: "NYSE",
  currency: "USD" as const,
};
const etf = {
  id: "asset-etf",
  symbol: "ACMX",
  name: "Acme Index Fund",
  assetType: "ETF" as const,
  exchange: "NYSEARCA",
  currency: "USD" as const,
};
const crypto = {
  id: "asset-crypto",
  symbol: "BTC",
  name: "Bitcoin",
  assetType: "CRYPTO" as const,
  exchange: "CRYPTO",
  currency: "USD" as const,
};

function queryState(overrides = {}) {
  return {
    isLoading: false,
    isError: false,
    isSuccess: true,
    data: { pages: [{ items: [], nextCursor: null }] },
    hasNextPage: false,
    isFetchingNextPage: false,
    error: null,
    refetch: vi.fn(),
    fetchNextPage: vi.fn(),
    ...overrides,
  };
}

describe("AssetDiscoveryScreen", () => {
  beforeEach(() => useAssets.mockReset());

  it("shows loading and both neutral empty states", () => {
    useAssets.mockReturnValue(
      queryState({ isLoading: true, isSuccess: false }),
    );
    const { rerender } = render(<AssetDiscoveryScreen />);
    expect(screen.getByRole("status")).toHaveTextContent("Loading assets…");
    useAssets.mockReturnValue(queryState());
    rerender(<AssetDiscoveryScreen />);
    expect(
      screen.getByText("No supported assets are available."),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Search assets"), {
      target: { value: "BTC" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    expect(
      screen.getByText("No assets match the current search and filter."),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(
      screen.getByText("No supported assets are available."),
    ).toBeInTheDocument();
  });

  it("renders canonical EQUITY, ETF, and CRYPTO metadata only", () => {
    useAssets.mockReturnValue(
      queryState({
        data: { pages: [{ items: [equity, etf, crypto], nextCursor: null }] },
      }),
    );
    render(<AssetDiscoveryScreen />);
    const assetList = within(screen.getByRole("list", { name: "Assets" }));
    for (const asset of [equity, etf, crypto]) {
      expect(assetList.getByText(asset.symbol)).toBeInTheDocument();
      expect(assetList.getByText(asset.name)).toBeInTheDocument();
      expect(assetList.getAllByText(asset.assetType).length).toBeGreaterThan(0);
      expect(assetList.getAllByText(asset.exchange).length).toBeGreaterThan(0);
      expect(assetList.getAllByText(asset.currency).length).toBeGreaterThan(0);
    }
    expect(
      assetList.queryByText(/Price|Holdings|Valuation|P\/L/),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Add|Edit|Delete Asset/ }),
    ).not.toBeInTheDocument();
  });

  it("applies search and exact AssetType filters without hidden normalization", () => {
    useAssets.mockReturnValue(queryState());
    render(<AssetDiscoveryScreen />);
    fireEvent.change(screen.getByLabelText("Search assets"), {
      target: { value: " BTC " },
    });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    expect(useAssets).toHaveBeenLastCalledWith({
      search: " BTC ",
      assetType: undefined,
      limit: 25,
    });
    for (const [label, assetType] of [
      ["Equity", "EQUITY"],
      ["ETF", "ETF"],
      ["Crypto", "CRYPTO"],
    ] as const) {
      fireEvent.click(screen.getByRole("button", { name: label }));
      expect(useAssets).toHaveBeenLastCalledWith({
        search: " BTC ",
        assetType,
        limit: 25,
      });
    }
    fireEvent.click(screen.getByRole("button", { name: "All" }));
    expect(useAssets).toHaveBeenLastCalledWith({
      search: " BTC ",
      assetType: undefined,
      limit: 25,
    });
    fireEvent.click(screen.getByRole("button", { name: "Clear search" }));
    expect(useAssets).toHaveBeenLastCalledWith({
      search: undefined,
      assetType: undefined,
      limit: 25,
    });
  });

  it("rejects search over 100 Unicode code points before updating the query", async () => {
    useAssets.mockReturnValue(queryState());
    render(<AssetDiscoveryScreen />);
    fireEvent.change(screen.getByLabelText("Search assets"), {
      target: { value: "😀".repeat(101) },
    });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Search must be 100 characters or fewer.",
    );
    expect(useAssets).toHaveBeenLastCalledWith({
      search: undefined,
      assetType: undefined,
      limit: 25,
    });
  });

  it("loads the next opaque page and hides Load more when no page remains", async () => {
    const fetchNextPage = vi.fn();
    useAssets.mockReturnValue(
      queryState({
        data: { pages: [{ items: [equity], nextCursor: "opaque" }] },
        hasNextPage: true,
        fetchNextPage,
      }),
    );
    const { rerender } = render(<AssetDiscoveryScreen />);
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    expect(fetchNextPage).toHaveBeenCalledTimes(1);
    useAssets.mockReturnValue(
      queryState({
        data: { pages: [{ items: [equity, etf], nextCursor: null }] },
      }),
    );
    rerender(<AssetDiscoveryScreen />);
    expect(
      screen.queryByRole("button", { name: /Load more/ }),
    ).not.toBeInTheDocument();
  });

  it("shows a safe error, correlation reference, and retry", () => {
    const refetch = vi.fn();
    useAssets.mockReturnValue(
      queryState({
        isError: true,
        isSuccess: false,
        error: new AssetApiError(500, "INTERNAL_ERROR", "details", "cid-asset"),
        refetch,
      }),
    );
    render(<AssetDiscoveryScreen />);
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Asset data is temporarily unavailable. Please try again.",
    );
    expect(screen.getByText("Reference: cid-asset")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
