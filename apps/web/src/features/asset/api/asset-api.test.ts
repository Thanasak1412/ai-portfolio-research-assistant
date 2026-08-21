import { afterEach, describe, expect, it, vi } from "vitest";

import { AssetApiError, assetApi } from "@/features/asset/api/asset-api";
import { isAccessTokenInvalid } from "@/platform/api/api-error";

const equity = {
  id: "asset_equity",
  symbol: "ACME",
  name: "Acme Corporation",
  assetType: "EQUITY" as const,
  exchange: "NYSE",
  currency: "USD" as const,
};
const etf = {
  id: "asset_etf",
  symbol: "ACMX",
  name: "Acme Index Fund",
  assetType: "ETF" as const,
  exchange: "NYSEARCA",
  currency: "USD" as const,
};
const crypto = {
  id: "asset_crypto",
  symbol: "BTC",
  name: "Bitcoin",
  assetType: "CRYPTO" as const,
  exchange: "CRYPTO",
  currency: "USD" as const,
};

afterEach(() => vi.unstubAllGlobals());

function mockFetch(response: Response) {
  const fetchMock = vi.fn().mockResolvedValue(response);
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

describe("Asset API", () => {
  it("uses Bearer authentication and omits credentials for an unfiltered request", async () => {
    const fetchMock = mockFetch(
      new Response(JSON.stringify({ items: [equity], nextCursor: null }), {
        status: 200,
      }),
    );
    await expect(assetApi.list("memory-token", {})).resolves.toEqual({
      items: [equity],
      nextCursor: null,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/assets",
      expect.objectContaining({
        credentials: "omit",
        headers: {
          Accept: "application/json",
          Authorization: "Bearer memory-token",
        },
      }),
    );
  });

  it("encodes search and sends exact frozen filter and cursor values", async () => {
    const fetchMock = mockFetch(
      new Response(
        JSON.stringify({ items: [crypto], nextCursor: "opaque+/=" }),
        {
          status: 200,
        },
      ),
    );
    await assetApi.list("memory-token", {
      search: "BTC & coin",
      assetType: "CRYPTO",
      cursor: "opaque cursor+/=",
      limit: 25,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/assets?search=BTC+%26+coin&type=CRYPTO&cursor=opaque+cursor%2B%2F%3D&limit=25",
      expect.anything(),
    );
    mockFetch(
      new Response(JSON.stringify({ items: [etf], nextCursor: null }), {
        status: 200,
      }),
    );
    await assetApi.list("memory-token", { search: "", assetType: "ETF" });
    expect(fetch).toHaveBeenLastCalledWith(
      "/api/v1/assets?type=ETF",
      expect.anything(),
    );
  });

  it("accepts the only supported canonical Asset variants", async () => {
    mockFetch(
      new Response(
        JSON.stringify({ items: [equity, etf, crypto], nextCursor: null }),
        {
          status: 200,
        },
      ),
    );
    await expect(assetApi.list("memory-token", {})).resolves.toEqual({
      items: [equity, etf, crypto],
      nextCursor: null,
    });
  });

  it("fails closed for malformed catalog data", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          items: [{ ...crypto, exchange: "COINBASE" }],
          nextCursor: null,
        }),
        { status: 200, headers: { "X-Correlation-ID": "cid-malformed" } },
      ),
    );
    await expect(assetApi.list("memory-token", {})).rejects.toMatchObject({
      status: 200,
      code: "INTERNAL_ERROR",
      correlationId: "cid-malformed",
    });
    mockFetch(
      new Response(
        JSON.stringify({
          items: [{ ...equity, currency: "EUR" }],
          nextCursor: null,
        }),
        { status: 200 },
      ),
    );
    await expect(assetApi.list("memory-token", {})).rejects.toMatchObject({
      code: "INTERNAL_ERROR",
    });
    mockFetch(
      new Response(
        JSON.stringify({
          items: [{ ...equity, assetType: "BOND" }],
          nextCursor: null,
        }),
        { status: 200 },
      ),
    );
    await expect(assetApi.list("memory-token", {})).rejects.toMatchObject({
      code: "INTERNAL_ERROR",
    });
    mockFetch(
      new Response(JSON.stringify({ items: [equity], nextCursor: "" }), {
        status: 200,
      }),
    );
    await expect(assetApi.list("memory-token", {})).rejects.toMatchObject({
      code: "INTERNAL_ERROR",
    });
  });

  it("preserves safe API errors and access-token classification", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          error: {
            code: "INVALID_REQUEST",
            message: "The request is invalid",
            correlationId: "cid-400",
          },
        }),
        { status: 400 },
      ),
    );
    await expect(assetApi.list("memory-token", {})).rejects.toMatchObject({
      status: 400,
      code: "INVALID_REQUEST",
      correlationId: "cid-400",
    });
    mockFetch(
      new Response(
        JSON.stringify({
          error: {
            code: "ACCESS_TOKEN_INVALID",
            message: "The access token is invalid",
            correlationId: "cid-401",
          },
        }),
        { status: 401 },
      ),
    );
    await expect(assetApi.list("memory-token", {})).rejects.toSatisfy(
      (error: unknown) =>
        error instanceof AssetApiError && isAccessTokenInvalid(error),
    );
    mockFetch(new Response("not-json", { status: 500 }));
    await expect(assetApi.list("memory-token", {})).rejects.toMatchObject({
      status: 500,
      code: "INTERNAL_ERROR",
    });
  });

  it("collapses network failures to a safe Asset service error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network")));
    await expect(assetApi.list("memory-token", {})).rejects.toMatchObject({
      status: 0,
      code: "INTERNAL_ERROR",
    });
  });
});
