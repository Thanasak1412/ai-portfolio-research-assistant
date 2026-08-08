import { afterEach, describe, expect, it, vi } from "vitest";

import { authApi } from "@/features/auth/api/auth-api";

const validCredential = "x".repeat(12);

const session = {
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

afterEach(() => vi.unstubAllGlobals());

function mockFetch(response: Response) {
  const fetchMock = vi.fn().mockResolvedValue(response);
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

describe("Authentication API", () => {
  it("uses same-origin JSON for registration and login", async () => {
    const fetchMock = mockFetch(
      new Response(JSON.stringify(session), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await authApi.register({
      email: "person@example.com",
      password: validCredential,
    });
    expect(fetchMock).toHaveBeenLastCalledWith(
      "/api/v1/auth/register",
      expect.objectContaining({
        method: "POST",
        credentials: "same-origin",
        body: JSON.stringify({
          email: "person@example.com",
          password: validCredential,
        }),
      }),
    );
    mockFetch(
      new Response(JSON.stringify(session), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await authApi.login({
      email: "person@example.com",
      password: validCredential,
    });
    expect(fetch).toHaveBeenLastCalledWith(
      "/api/v1/auth/login",
      expect.objectContaining({ method: "POST", credentials: "same-origin" }),
    );
  });

  it("parses generic backend errors without exposing a response object", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          error: {
            code: "AUTHENTICATION_FAILED",
            message: "Authentication failed",
            correlationId: "cid",
          },
        }),
        { status: 401, headers: { "Content-Type": "application/json" } },
      ),
    );
    await expect(
      authApi.login({
        email: "person@example.com",
        password: validCredential,
      }),
    ).rejects.toMatchObject({
      status: 401,
      code: "AUTHENTICATION_FAILED",
      correlationId: "cid",
    });
  });

  it("uses cookie-owned refresh and logout primitives without a request body", async () => {
    const refreshFetch = mockFetch(
      new Response(
        JSON.stringify({
          accessToken: "replacement",
          tokenType: "Bearer",
          expiresIn: 900,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    await authApi.refresh();
    expect(refreshFetch).toHaveBeenCalledWith(
      "/api/v1/auth/refresh",
      expect.objectContaining({
        method: "POST",
        credentials: "same-origin",
        headers: expect.objectContaining({
          "X-Requested-With": "portfolio-web",
        }),
      }),
    );
    expect(refreshFetch.mock.calls[0]?.[1]?.body).toBeUndefined();
    const logoutFetch = mockFetch(new Response(null, { status: 204 }));
    await authApi.logout();
    expect(logoutFetch).toHaveBeenCalledWith(
      "/api/v1/auth/logout",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          "X-Requested-With": "portfolio-web",
        }),
      }),
    );
  });

  it("sends an explicit opaque bearer token only to me", async () => {
    const fetchMock = mockFetch(
      new Response(JSON.stringify(session.user), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await authApi.me("opaque-token");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/auth/me",
      expect.objectContaining({
        credentials: "same-origin",
        headers: expect.objectContaining({
          Authorization: "Bearer opaque-token",
        }),
      }),
    );
  });

  it("validates a positive Retry-After value", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          error: {
            code: "RATE_LIMIT_EXCEEDED",
            message: "Limited",
            correlationId: "cid",
          },
        }),
        {
          status: 429,
          headers: { "Content-Type": "application/json", "Retry-After": "15" },
        },
      ),
    );
    await expect(
      authApi.login({
        email: "person@example.com",
        password: validCredential,
      }),
    ).rejects.toMatchObject({
      code: "RATE_LIMIT_EXCEEDED",
      retryAfterSeconds: 15,
    });
  });
});
