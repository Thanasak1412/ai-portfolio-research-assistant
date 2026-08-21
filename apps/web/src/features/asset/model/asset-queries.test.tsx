import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { assetKeys } from "@/features/asset/model/asset-query-keys";
import { useAssets } from "@/features/asset/model/asset-queries";

const assetApiSpies = vi.hoisted(() => ({ list: vi.fn() }));
const authSpies = vi.hoisted(() => ({
  runAuthenticated: vi.fn((operation: (token: string) => unknown) =>
    operation("memory-token"),
  ),
}));

vi.mock("@/features/asset/api/asset-api", async (importOriginal) => {
  const original =
    await importOriginal<typeof import("@/features/asset/api/asset-api")>();
  return {
    ...original,
    assetApi: { ...original.assetApi, ...assetApiSpies },
  };
});
vi.mock("@/features/auth/model/auth-session-provider", () => ({
  useAuthSession: () => ({
    state: { status: "authenticated" },
    runAuthenticated: authSpies.runAuthenticated,
  }),
}));

const first = {
  id: "asset-a",
  symbol: "AAA",
  name: "Alpha",
  assetType: "EQUITY" as const,
  exchange: "NYSE",
  currency: "USD" as const,
};
const second = {
  id: "asset-b",
  symbol: "BBB",
  name: "Beta",
  assetType: "ETF" as const,
  exchange: "NASDAQ",
  currency: "USD" as const,
};

function Probe({
  search,
  assetType,
}: Readonly<{ search?: string; assetType?: "EQUITY" | "ETF" | "CRYPTO" }>) {
  const result = useAssets({ search, assetType, limit: 25 });
  return (
    <div>
      <output>
        {result.data?.pages
          .flatMap((page) => page.items)
          .map((asset) => asset.symbol)
          .join(",")}
      </output>
      <button type="button" onClick={() => void result.fetchNextPage()}>
        next
      </button>
      <output>{String(result.hasNextPage)}</output>
    </div>
  );
}

function renderProbe(props: {
  search?: string;
  assetType?: "EQUITY" | "ETF" | "CRYPTO";
}) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <Probe {...props} />
    </QueryClientProvider>,
  );
}

describe("Asset React Query discovery", () => {
  beforeEach(() => {
    assetApiSpies.list.mockReset();
    authSpies.runAuthenticated.mockClear();
  });

  it("uses the opaque cursor unchanged and preserves backend page order", async () => {
    assetApiSpies.list
      .mockResolvedValueOnce({ items: [first], nextCursor: "opaque+/=" })
      .mockResolvedValueOnce({ items: [second], nextCursor: null });
    renderProbe({ search: "Alpha", assetType: "EQUITY" });
    await waitFor(() =>
      expect(assetApiSpies.list).toHaveBeenCalledWith("memory-token", {
        search: "Alpha",
        assetType: "EQUITY",
        limit: 25,
        cursor: undefined,
      }),
    );
    fireEvent.click(screen.getByRole("button", { name: "next" }));
    await waitFor(() =>
      expect(assetApiSpies.list).toHaveBeenLastCalledWith("memory-token", {
        search: "Alpha",
        assetType: "EQUITY",
        limit: 25,
        cursor: "opaque+/=",
      }),
    );
    expect(screen.getByText("AAA,BBB")).toBeInTheDocument();
    expect(screen.getByText("false")).toBeInTheDocument();
  });

  it("uses independent filter keys and restarts without a previous cursor", async () => {
    assetApiSpies.list.mockResolvedValue({ items: [], nextCursor: null });
    const { rerender } = renderProbe({ search: "Alpha" });
    await waitFor(() => expect(assetApiSpies.list).toHaveBeenCalledTimes(1));
    rerender(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <Probe search="Beta" assetType="ETF" />
      </QueryClientProvider>,
    );
    await waitFor(() =>
      expect(assetApiSpies.list).toHaveBeenLastCalledWith("memory-token", {
        search: "Beta",
        assetType: "ETF",
        limit: 25,
        cursor: undefined,
      }),
    );
    expect(assetKeys.list({ search: "Alpha", limit: 25 })).not.toEqual(
      assetKeys.list({ search: "Beta", assetType: "ETF", limit: 25 }),
    );
  });
});
