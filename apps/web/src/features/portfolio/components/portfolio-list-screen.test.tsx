import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PortfolioListScreen } from "@/features/portfolio/components/portfolio-list-screen";
import { PortfolioApiError } from "@/features/portfolio/api/portfolio-api";

const push = vi.fn();
const usePortfolios = vi.fn();
const useCreatePortfolio = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));
vi.mock("@/features/portfolio/model/portfolio-queries", () => ({
  usePortfolios: (...args: unknown[]) => usePortfolios(...args),
  useCreatePortfolio: () => useCreatePortfolio(),
}));

const activePortfolio = {
  id: "portfolio-1",
  name: "Growth",
  baseCurrency: "USD" as const,
  status: "ACTIVE" as const,
  archivedAt: null,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-02T00:00:00Z",
};

function renderScreen() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <PortfolioListScreen />
    </QueryClientProvider>,
  );
}
function queryState(overrides = {}) {
  return {
    isLoading: false,
    isError: false,
    isSuccess: true,
    data: { items: [] },
    error: null,
    refetch: vi.fn(),
    ...overrides,
  };
}

describe("PortfolioListScreen", () => {
  beforeEach(() => {
    push.mockReset();
    useCreatePortfolio.mockReturnValue({
      isPending: false,
      mutateAsync: vi.fn(),
    });
  });

  it("distinguishes loading, active empty, archived empty, and server-selected status", () => {
    usePortfolios.mockReturnValue(
      queryState({ isLoading: true, isSuccess: false }),
    );
    const { rerender } = renderScreen();
    expect(screen.getByRole("status")).toHaveTextContent("Loading portfolios…");
    usePortfolios.mockReturnValue(queryState());
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <PortfolioListScreen />
      </QueryClientProvider>,
    );
    expect(screen.getByText("No active portfolios yet.")).toBeInTheDocument();
    expect(screen.getByLabelText("Portfolio name")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Archived" }));
    expect(usePortfolios).toHaveBeenLastCalledWith("ARCHIVED");
    expect(screen.getByText("No archived portfolios.")).toBeInTheDocument();
    expect(screen.queryByLabelText("Portfolio name")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Active" }));
    expect(screen.getByLabelText("Portfolio name")).toBeInTheDocument();
  });

  it("renders only neutral metadata and retries a failed query", () => {
    const refetch = vi.fn();
    usePortfolios.mockReturnValue(
      queryState({ data: { items: [activePortfolio] }, refetch }),
    );
    const { rerender } = renderScreen();
    expect(screen.getByRole("link", { name: /Growth/ })).toHaveAttribute(
      "href",
      "/app/portfolios/portfolio-1",
    );
    expect(
      screen.queryByText(/Market Value|Holdings|P\/L/),
    ).not.toBeInTheDocument();
    usePortfolios.mockReturnValue(
      queryState({
        isError: true,
        isSuccess: false,
        error: new Error("down"),
        refetch,
      }),
    );
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <PortfolioListScreen />
      </QueryClientProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("creates an active USD Portfolio and routes using the authoritative response", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(activePortfolio);
    useCreatePortfolio.mockReturnValue({ isPending: false, mutateAsync });
    usePortfolios.mockReturnValue(queryState());
    renderScreen();
    fireEvent.change(screen.getByLabelText("Portfolio name"), {
      target: { value: "Growth" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create Portfolio" }));
    await waitFor(() =>
      expect(mutateAsync).toHaveBeenCalledWith({
        name: "Growth",
        baseCurrency: "USD",
      }),
    );
    expect(push).toHaveBeenCalledWith("/app/portfolios/portfolio-1");
  });

  it("uses accessible client validation before create", async () => {
    usePortfolios.mockReturnValue(queryState());
    renderScreen();
    fireEvent.click(screen.getByRole("button", { name: "Create Portfolio" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Portfolio name is required.",
    );
  });

  it("shows the safe duplicate-name response without locally reserving names", async () => {
    const mutateAsync = vi
      .fn()
      .mockRejectedValue(
        new PortfolioApiError(409, "PORTFOLIO_NAME_CONFLICT", "Conflict"),
      );
    useCreatePortfolio.mockReturnValue({ isPending: false, mutateAsync });
    usePortfolios.mockReturnValue(queryState());
    renderScreen();
    fireEvent.change(screen.getByLabelText("Portfolio name"), {
      target: { value: "Growth" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create Portfolio" }));
    expect(
      await screen.findByText(
        "An active portfolio with this name already exists.",
      ),
    ).toBeInTheDocument();
  });
});
