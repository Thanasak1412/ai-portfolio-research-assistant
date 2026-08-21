import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { AuthApiError, type AuthApi } from "@/features/auth/api/auth-api";
import {
  AuthSessionProvider,
  useAuthSession,
} from "@/features/auth/model/auth-session-provider";
import {
  useArchivePortfolio,
  useCreatePortfolio,
  useUpdatePortfolio,
} from "@/features/portfolio/model/portfolio-queries";

const portfolioApiSpies = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
  archive: vi.fn(),
}));
vi.mock("@/features/portfolio/api/portfolio-api", async (importOriginal) => {
  const original =
    await importOriginal<
      typeof import("@/features/portfolio/api/portfolio-api")
    >();
  return {
    ...original,
    portfolioApi: { ...original.portfolioApi, ...portfolioApiSpies },
  };
});

const access = {
  accessToken: "memory-token",
  tokenType: "Bearer" as const,
  expiresIn: 900 as const,
};
const session = {
  ...access,
  user: {
    id: "user-1",
    email: "person@example.test",
    status: "active" as const,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  },
};
const active = {
  id: "portfolio-1",
  name: "Growth",
  baseCurrency: "USD" as const,
  status: "ACTIVE" as const,
  archivedAt: null,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};
const archived = {
  ...active,
  status: "ARCHIVED" as const,
  archivedAt: "2026-01-02T00:00:00Z",
};

function rejectedBootstrapApi(): AuthApi {
  return {
    register: vi.fn(),
    login: vi.fn(),
    refresh: vi
      .fn()
      .mockRejectedValue(
        new AuthApiError(401, "SESSION_REFRESH_REJECTED", "rejected"),
      ),
    logout: vi.fn(),
    me: vi.fn(),
  };
}

function MutationProbe() {
  const { establishSession, state } = useAuthSession();
  const createMutation = useCreatePortfolio();
  const updateMutation = useUpdatePortfolio();
  const archiveMutation = useArchivePortfolio();
  return (
    <div>
      <output>{state.status}</output>
      <button
        onClick={() => {
          establishSession(session);
          void createMutation.mutateAsync({
            name: "Growth",
            baseCurrency: "USD",
          });
        }}
      >
        create
      </button>
      <button
        onClick={() => {
          establishSession(session);
          void updateMutation.mutateAsync({
            portfolioId: active.id,
            input: { name: "New Growth" },
          });
        }}
      >
        update
      </button>
      <button
        onClick={() => {
          establishSession(session);
          void archiveMutation.mutateAsync(active.id);
        }}
      >
        archive
      </button>
    </div>
  );
}

describe("Portfolio React Query mutations", () => {
  it("uses backend responses to update detail state and invalidate the required lists", async () => {
    portfolioApiSpies.create.mockResolvedValue(active);
    portfolioApiSpies.update.mockResolvedValue({
      ...active,
      name: "New Growth",
    });
    portfolioApiSpies.archive.mockResolvedValue(archived);
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidate = vi.spyOn(client, "invalidateQueries");
    render(
      <QueryClientProvider client={client}>
        <AuthSessionProvider api={rejectedBootstrapApi()}>
          <MutationProbe />
        </AuthSessionProvider>
      </QueryClientProvider>,
    );
    await screen.findByText("unauthenticated");
    fireEvent.click(screen.getByRole("button", { name: "create" }));
    await waitFor(() =>
      expect(portfolioApiSpies.create).toHaveBeenCalledWith("memory-token", {
        name: "Growth",
        baseCurrency: "USD",
      }),
    );
    expect(client.getQueryData(["portfolios", "detail", active.id])).toEqual(
      active,
    );
    fireEvent.click(screen.getByRole("button", { name: "update" }));
    await waitFor(() =>
      expect(portfolioApiSpies.update).toHaveBeenCalledWith(
        "memory-token",
        active.id,
        {
          name: "New Growth",
        },
      ),
    );
    fireEvent.click(screen.getByRole("button", { name: "archive" }));
    await waitFor(() =>
      expect(portfolioApiSpies.archive).toHaveBeenCalledWith(
        "memory-token",
        active.id,
      ),
    );
    expect(client.getQueryData(["portfolios", "detail", active.id])).toEqual(
      archived,
    );
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: ["portfolios", "list", "ACTIVE"],
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: ["portfolios", "list", "ARCHIVED"],
    });
  });
});
