import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PortfolioApiError } from "@/features/portfolio/api/portfolio-api";
import { PortfolioDetailScreen } from "@/features/portfolio/components/portfolio-detail-screen";

const usePortfolio = vi.fn();
const useUpdatePortfolio = vi.fn();
const useArchivePortfolio = vi.fn();

vi.mock("@/features/portfolio/model/portfolio-queries", () => ({
  usePortfolio: (...args: unknown[]) => usePortfolio(...args),
  useUpdatePortfolio: () => useUpdatePortfolio(),
  useArchivePortfolio: () => useArchivePortfolio(),
}));

const active = {
  id: "portfolio-1",
  name: "Growth",
  baseCurrency: "USD" as const,
  status: "ACTIVE" as const,
  archivedAt: null,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-02T00:00:00Z",
};
const archived = {
  ...active,
  status: "ARCHIVED" as const,
  archivedAt: "2026-01-03T00:00:00Z",
};

function queryState(overrides = {}) {
  return {
    isLoading: false,
    isError: false,
    data: active,
    error: null,
    refetch: vi.fn(),
    ...overrides,
  };
}
function renderScreen() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <PortfolioDetailScreen portfolioId="portfolio-1" />
    </QueryClientProvider>,
  );
}

describe("PortfolioDetailScreen", () => {
  beforeEach(() => {
    useUpdatePortfolio.mockReturnValue({
      isPending: false,
      mutateAsync: vi.fn(),
    });
    useArchivePortfolio.mockReturnValue({
      isPending: false,
      mutateAsync: vi.fn(),
    });
  });

  it("renders loading, safe not-found, and retryable error states", () => {
    usePortfolio.mockReturnValue(
      queryState({ isLoading: true, data: undefined }),
    );
    const { rerender } = renderScreen();
    expect(screen.getByRole("status")).toHaveTextContent("Loading portfolio…");
    usePortfolio.mockReturnValue(
      queryState({
        isError: true,
        data: undefined,
        error: new PortfolioApiError(404, "PORTFOLIO_NOT_FOUND", "Missing"),
      }),
    );
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <PortfolioDetailScreen portfolioId="portfolio-1" />
      </QueryClientProvider>,
    );
    expect(screen.getByText("Portfolio not found.")).toBeInTheDocument();
    const refetch = vi.fn();
    usePortfolio.mockReturnValue(
      queryState({
        isError: true,
        data: undefined,
        error: new Error("down"),
        refetch,
      }),
    );
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <PortfolioDetailScreen portfolioId="portfolio-1" />
      </QueryClientProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("allows active rename and archive confirmation without financial controls", async () => {
    const update = vi.fn().mockResolvedValue({ ...active, name: "New Growth" });
    const archive = vi.fn().mockResolvedValue(archived);
    useUpdatePortfolio.mockReturnValue({
      isPending: false,
      mutateAsync: update,
    });
    useArchivePortfolio.mockReturnValue({
      isPending: false,
      mutateAsync: archive,
    });
    usePortfolio.mockReturnValue(queryState());
    renderScreen();
    fireEvent.change(screen.getByLabelText("Portfolio name"), {
      target: { value: "New Growth" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save name" }));
    await waitFor(() =>
      expect(update).toHaveBeenCalledWith({
        portfolioId: "portfolio-1",
        input: { name: "New Growth" },
      }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Archive Portfolio" }));
    expect(screen.getByText(/record is retained/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.queryByText(/record is retained/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Archive Portfolio" }));
    fireEvent.click(screen.getByRole("button", { name: "Archive Portfolio" }));
    await waitFor(() => expect(archive).toHaveBeenCalledWith("portfolio-1"));
    expect(
      screen.queryByText(/Market Value|Holdings|P\/L/),
    ).not.toBeInTheDocument();
  });

  it("renders archived Portfolios read-only without mutation controls", () => {
    usePortfolio.mockReturnValue(queryState({ data: archived }));
    renderScreen();
    expect(screen.getByText("Archived Portfolio")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Save name" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Archive Portfolio" }),
    ).not.toBeInTheDocument();
  });

  it("shows rename conflicts and refetches after an archived-state race", async () => {
    const conflict = vi
      .fn()
      .mockRejectedValueOnce(
        new PortfolioApiError(409, "PORTFOLIO_NAME_CONFLICT", "Conflict"),
      )
      .mockRejectedValueOnce(
        new PortfolioApiError(422, "PORTFOLIO_ARCHIVED", "Archived"),
      );
    useUpdatePortfolio.mockReturnValue({
      isPending: false,
      mutateAsync: conflict,
    });
    usePortfolio.mockReturnValue(queryState());
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    render(
      <QueryClientProvider client={client}>
        <PortfolioDetailScreen portfolioId="portfolio-1" />
      </QueryClientProvider>,
    );
    const input = screen.getByLabelText("Portfolio name");
    fireEvent.change(input, { target: { value: "Duplicate" } });
    fireEvent.click(screen.getByRole("button", { name: "Save name" }));
    expect(
      await screen.findByText(
        "An active portfolio with this name already exists.",
      ),
    ).toBeInTheDocument();
    fireEvent.change(input, { target: { value: "Archived race" } });
    fireEvent.click(screen.getByRole("button", { name: "Save name" }));
    expect(
      await screen.findByText(
        "This portfolio is archived and can no longer be edited.",
      ),
    ).toBeInTheDocument();
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: ["portfolios", "detail", "portfolio-1"],
    });
  });
});
