import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  AuthSessionProvider,
  useAuthSession,
} from "@/features/auth/model/auth-session-provider";

const sessionResponse = {
  accessToken: "opaque-token",
  tokenType: "Bearer" as const,
  expiresIn: 900 as const,
  user: {
    id: "user_opaque",
    email: "person@example.com",
    status: "active" as const,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  },
};
function StateProbe() {
  const {
    session: authSession,
    establishSession,
    replaceAccessToken,
    clearMemorySession,
  } = useAuthSession();
  return (
    <>
      <output>{authSession?.user.email ?? "anonymous"}</output>
      <button onClick={() => establishSession(sessionResponse)}>
        establish
      </button>
      <button
        onClick={() =>
          replaceAccessToken({
            accessToken: "replacement",
            tokenType: "Bearer",
            expiresIn: 900,
          })
        }
      >
        replace
      </button>
      <button onClick={clearMemorySession}>clear</button>
    </>
  );
}

describe("AuthSessionProvider", () => {
  it("keeps only in-memory state and starts each provider empty", async () => {
    const { unmount } = render(
      <AuthSessionProvider>
        <StateProbe />
      </AuthSessionProvider>,
    );
    expect(screen.getByText("anonymous")).toBeInTheDocument();
    fireEvent.click(screen.getByText("establish"));
    expect(await screen.findByText("person@example.com")).toBeInTheDocument();
    fireEvent.click(screen.getByText("replace"));
    fireEvent.click(screen.getByText("clear"));
    await waitFor(() =>
      expect(screen.getByText("anonymous")).toBeInTheDocument(),
    );
    unmount();
    render(
      <AuthSessionProvider>
        <StateProbe />
      </AuthSessionProvider>,
    );
    expect(screen.getByText("anonymous")).toBeInTheDocument();
    expect(window.localStorage.length).toBe(0);
    expect(window.sessionStorage.length).toBe(0);
  });
});
