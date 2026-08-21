import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ProtectedApplicationShell } from "@/features/auth/components/protected-application-shell";

vi.mock("@/features/auth/model/auth-session-provider", () => ({
  useAuthSession: () => ({
    state: {
      status: "authenticated",
      session: { user: { email: "person@example.test" } },
    },
  }),
}));
vi.mock("@/features/auth/components/logout-button", () => ({
  LogoutButton: () => <button type="button">Logout</button>,
}));

describe("ProtectedApplicationShell", () => {
  it("keeps the M2 application navigation focused on Home, Portfolios, and Assets", () => {
    render(
      <ProtectedApplicationShell>
        <p>Protected content</p>
      </ProtectedApplicationShell>,
    );
    expect(screen.getByRole("link", { name: "Home" })).toHaveAttribute(
      "href",
      "/app",
    );
    expect(screen.getByRole("link", { name: "Portfolios" })).toHaveAttribute(
      "href",
      "/app/portfolios",
    );
    expect(screen.getByRole("link", { name: "Assets" })).toHaveAttribute(
      "href",
      "/app/assets",
    );
    expect(
      screen.queryByRole("link", {
        name: /Transactions|Holdings|Prices|Dashboard|AI/,
      }),
    ).not.toBeInTheDocument();
  });
});
