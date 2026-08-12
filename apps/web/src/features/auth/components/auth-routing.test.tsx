import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AuthApi } from "@/features/auth/api/auth-api";
import { AuthApiError } from "@/features/auth/api/auth-api";
import { LogoutButton } from "@/features/auth/components/logout-button";
import { PublicAuthRoute } from "@/features/auth/components/public-auth-route";
import { RequireAuthenticated } from "@/features/auth/components/require-authenticated";
import { AuthSessionProvider } from "@/features/auth/model/auth-session-provider";

const replace = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));

const access = {
  accessToken: "memory-access",
  tokenType: "Bearer" as const,
  expiresIn: 900 as const,
};
const user = {
  id: "user_opaque",
  email: "person@example.com",
  status: "active" as const,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};

function api(overrides: Partial<AuthApi> = {}): AuthApi {
  return {
    register: vi.fn(),
    login: vi.fn(),
    refresh: vi
      .fn()
      .mockRejectedValue(
        new AuthApiError(401, "SESSION_REFRESH_REJECTED", "rejected"),
      ),
    logout: vi.fn().mockResolvedValue(undefined),
    me: vi.fn(),
    ...overrides,
  };
}

describe("Authentication routing", () => {
  beforeEach(() => replace.mockReset());

  it("shows no protected content during bootstrap and redirects only after rejection", async () => {
    let rejectRefresh: ((reason: unknown) => void) | undefined;
    const refresh = vi.fn(
      () =>
        new Promise<typeof access>((_resolve, reject) => {
          rejectRefresh = reject;
        }),
    );
    render(
      <AuthSessionProvider api={api({ refresh })}>
        <RequireAuthenticated>
          <p>protected content</p>
        </RequireAuthenticated>
      </AuthSessionProvider>,
    );
    expect(
      screen.getByText("Restoring your secure session…"),
    ).toBeInTheDocument();
    expect(screen.queryByText("protected content")).not.toBeInTheDocument();
    expect(replace).not.toHaveBeenCalled();
    rejectRefresh?.(
      new AuthApiError(401, "SESSION_REFRESH_REJECTED", "rejected"),
    );
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
    expect(screen.queryByText("protected content")).not.toBeInTheDocument();
  });

  it("renders protected content after authoritative bootstrap and redirects auth pages", async () => {
    const authenticatedApi = api({
      refresh: vi.fn().mockResolvedValue(access),
      me: vi.fn().mockResolvedValue(user),
    });
    const { unmount } = render(
      <AuthSessionProvider api={authenticatedApi}>
        <RequireAuthenticated>
          <p>protected content</p>
        </RequireAuthenticated>
      </AuthSessionProvider>,
    );
    expect(await screen.findByText("protected content")).toBeInTheDocument();
    unmount();
    replace.mockReset();
    render(
      <AuthSessionProvider api={authenticatedApi}>
        <PublicAuthRoute>
          <p>login form</p>
        </PublicAuthRoute>
      </AuthSessionProvider>,
    );
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/app"));
    expect(screen.queryByText("login form")).not.toBeInTheDocument();
  });

  it("disables repeated logout and clears the UI even when revocation is unavailable", async () => {
    let rejectLogout: ((reason: unknown) => void) | undefined;
    const logout = vi.fn(
      () =>
        new Promise<void>((_resolve, reject) => {
          rejectLogout = reject;
        }),
    );
    render(
      <AuthSessionProvider
        api={api({
          refresh: vi.fn().mockResolvedValue(access),
          me: vi.fn().mockResolvedValue(user),
          logout,
        })}
      >
        <LogoutButton />
      </AuthSessionProvider>,
    );
    const button = await screen.findByRole("button", { name: "Sign out" });
    fireEvent.click(button);
    fireEvent.click(button);
    expect(logout).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: "Signing out…" })).toBeDisabled();
    rejectLogout?.(
      new AuthApiError(503, "AUTH_SERVICE_UNAVAILABLE", "unavailable"),
    );
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
  });
});
