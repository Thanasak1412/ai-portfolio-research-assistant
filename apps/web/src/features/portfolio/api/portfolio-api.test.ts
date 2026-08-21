import { afterEach, describe, expect, it, vi } from "vitest";

import {
  PortfolioApiError,
  portfolioApi,
} from "@/features/portfolio/api/portfolio-api";

const portfolio = {
  id: "portfolio_opaque/a",
  name: "Growth",
  baseCurrency: "USD" as const,
  status: "ACTIVE" as const,
  archivedAt: null,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};

afterEach(() => vi.unstubAllGlobals());

function mockFetch(response: Response) {
  const fetchMock = vi.fn().mockResolvedValue(response);
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

describe("Portfolio API", () => {
  it("lists ACTIVE and ARCHIVED records with a Bearer token", async () => {
    const fetchMock = mockFetch(
      new Response(JSON.stringify({ items: [portfolio] }), { status: 200 }),
    );
    await portfolioApi.list("memory-token", "ACTIVE");
    expect(fetchMock).toHaveBeenLastCalledWith(
      "/api/v1/portfolios?status=ACTIVE",
      expect.objectContaining({
        credentials: "omit",
        headers: expect.objectContaining({
          Authorization: "Bearer memory-token",
        }),
      }),
    );
    mockFetch(new Response(JSON.stringify({ items: [] }), { status: 200 }));
    await portfolioApi.list("memory-token", "ARCHIVED");
    expect(fetch).toHaveBeenLastCalledWith(
      "/api/v1/portfolios?status=ARCHIVED",
      expect.anything(),
    );
  });

  it("uses only frozen create, update, and archive fields", async () => {
    const fetchMock = mockFetch(
      new Response(JSON.stringify(portfolio), { status: 201 }),
    );
    await portfolioApi.create("memory-token", {
      name: "Growth",
      baseCurrency: "USD",
    });
    expect(fetchMock).toHaveBeenLastCalledWith(
      "/api/v1/portfolios",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ name: "Growth", baseCurrency: "USD" }),
      }),
    );
    mockFetch(new Response(JSON.stringify(portfolio), { status: 200 }));
    await portfolioApi.update("memory-token", "portfolio_opaque/a", {
      name: "New Growth",
    });
    expect(fetch).toHaveBeenLastCalledWith(
      "/api/v1/portfolios/portfolio_opaque%2Fa",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({ name: "New Growth" }),
      }),
    );
    mockFetch(
      new Response(
        JSON.stringify({
          ...portfolio,
          status: "ARCHIVED",
          archivedAt: "2026-01-02T00:00:00Z",
        }),
        { status: 200 },
      ),
    );
    await portfolioApi.archive("memory-token", portfolio.id);
    expect(fetch).toHaveBeenLastCalledWith(
      "/api/v1/portfolios/portfolio_opaque%2Fa/archive",
      expect.objectContaining({ method: "POST" }),
    );
    mockFetch(new Response(JSON.stringify(portfolio), { status: 200 }));
    await portfolioApi.get("memory-token", "portfolio_opaque/a");
    expect(fetch).toHaveBeenLastCalledWith(
      "/api/v1/portfolios/portfolio_opaque%2Fa",
      expect.anything(),
    );
  });

  it("parses safe public errors and preserves correlation information", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          error: {
            code: "PORTFOLIO_NAME_CONFLICT",
            message: "Conflict",
            correlationId: "cid",
          },
        }),
        { status: 409 },
      ),
    );
    await expect(
      portfolioApi.create("memory-token", {
        name: "Growth",
        baseCurrency: "USD",
      }),
    ).rejects.toMatchObject({
      status: 409,
      code: "PORTFOLIO_NAME_CONFLICT",
      correlationId: "cid",
    });
    mockFetch(
      new Response(
        JSON.stringify({
          error: {
            code: "PORTFOLIO_ARCHIVED",
            message: "Archived",
            correlationId: "cid2",
          },
        }),
        { status: 422 },
      ),
    );
    await expect(
      portfolioApi.update("memory-token", "id", { name: "New" }),
    ).rejects.toMatchObject({ code: "PORTFOLIO_ARCHIVED" });
    mockFetch(
      new Response(
        JSON.stringify({
          error: {
            code: "PORTFOLIO_NOT_FOUND",
            message: "Missing",
            correlationId: "cid3",
          },
        }),
        { status: 404 },
      ),
    );
    await expect(portfolioApi.get("memory-token", "id")).rejects.toMatchObject({
      code: "PORTFOLIO_NOT_FOUND",
    });
  });

  it("fails closed for malformed responses and keeps invalid token errors compatible", async () => {
    mockFetch(
      new Response(JSON.stringify({ items: [{ id: "only-id" }] }), {
        status: 200,
      }),
    );
    await expect(
      portfolioApi.list("memory-token", "ACTIVE"),
    ).rejects.toMatchObject({ code: "INTERNAL_ERROR" });
    mockFetch(
      new Response(
        JSON.stringify({
          error: {
            code: "ACCESS_TOKEN_INVALID",
            message: "Invalid",
            correlationId: "cid",
          },
        }),
        { status: 401 },
      ),
    );
    try {
      await portfolioApi.list("memory-token", "ACTIVE");
      throw new Error("expected ACCESS_TOKEN_INVALID");
    } catch (error) {
      expect(error).toBeInstanceOf(PortfolioApiError);
      expect(error).toMatchObject({
        status: 401,
        code: "ACCESS_TOKEN_INVALID",
      });
    }
  });
});
