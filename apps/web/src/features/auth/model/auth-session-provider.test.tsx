import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { StrictMode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AuthApi } from "@/features/auth/api/auth-api";
import { AuthApiError } from "@/features/auth/api/auth-api";
import type {
  AuthenticationSignal,
  BrowserSessionCoordinator,
} from "@/features/auth/model/auth-browser-coordinator";
import {
  AuthSessionProvider,
  useAuthSession,
} from "@/features/auth/model/auth-session-provider";

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
const sessionResponse = { ...access, user };

function rejectedSession() {
  return new AuthApiError(401, "SESSION_REFRESH_REJECTED", "rejected");
}

function invalidAccess() {
  return new AuthApiError(401, "ACCESS_TOKEN_INVALID", "invalid");
}

function fakeApi(overrides: Partial<AuthApi> = {}): AuthApi {
  return {
    register: vi.fn(),
    login: vi.fn(),
    refresh: vi.fn().mockRejectedValue(rejectedSession()),
    logout: vi.fn().mockResolvedValue(undefined),
    me: vi.fn(),
    ...overrides,
  };
}

function fakeCoordinator() {
  let listener: ((signal: AuthenticationSignal) => void) | undefined;
  const coordinator: BrowserSessionCoordinator & {
    locks: number;
    signals: AuthenticationSignal[];
    emit(signal: AuthenticationSignal): void;
  } = {
    locks: 0,
    signals: [],
    async withSessionTransition(operation) {
      coordinator.locks += 1;
      return operation();
    },
    broadcast(signal) {
      coordinator.signals.push(signal);
    },
    subscribe(next) {
      listener = next;
      return () => {
        listener = undefined;
      };
    },
    emit(signal) {
      listener?.(signal);
    },
  };
  return coordinator;
}

function StateProbe() {
  const { state, establishSession, runAuthenticated, logout, retryBootstrap } =
    useAuthSession();
  return (
    <>
      <output data-testid="status">{state.status}</output>
      <output data-testid="identity">
        {state.session?.user.email ?? "anonymous"}
      </output>
      <button onClick={() => establishSession(sessionResponse)}>
        establish
      </button>
      <button
        onClick={() =>
          void runAuthenticated(async (token) => {
            if (token === "memory-access") throw invalidAccess();
            return token;
          })
        }
      >
        protected
      </button>
      <button onClick={() => void logout()}>logout</button>
      <button onClick={retryBootstrap}>retry bootstrap</button>
    </>
  );
}

describe("AuthSessionProvider bootstrap", () => {
  beforeEach(() => {
    window.localStorage.clear();
    window.sessionStorage.clear();
  });

  it("starts bootstrapping, refreshes once, validates /me, and restores memory state", async () => {
    let resolveRefresh: ((value: typeof access) => void) | undefined;
    const refresh = vi.fn(
      () =>
        new Promise<typeof access>((resolve) => {
          resolveRefresh = resolve;
        }),
    );
    const me = vi.fn().mockResolvedValue(user);
    render(
      <AuthSessionProvider api={fakeApi({ refresh, me })}>
        <StateProbe />
      </AuthSessionProvider>,
    );
    expect(screen.getByTestId("status")).toHaveTextContent("bootstrapping");
    await act(async () => resolveRefresh?.(access));
    expect(await screen.findByText("authenticated")).toBeInTheDocument();
    expect(me).toHaveBeenCalledWith("memory-access");
    expect(screen.getByText("person@example.com")).toBeInTheDocument();
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(window.localStorage.length).toBe(0);
    expect(window.sessionStorage.length).toBe(0);
  });

  it("treats an absent session as unauthenticated and a service failure as recoverable", async () => {
    const { unmount } = render(
      <AuthSessionProvider api={fakeApi()}>
        <StateProbe />
      </AuthSessionProvider>,
    );
    expect(await screen.findByText("unauthenticated")).toBeInTheDocument();
    unmount();

    const refresh = vi
      .fn()
      .mockRejectedValueOnce(
        new AuthApiError(503, "AUTH_SERVICE_UNAVAILABLE", "unavailable"),
      )
      .mockResolvedValueOnce(access);
    render(
      <AuthSessionProvider
        api={fakeApi({ refresh, me: vi.fn().mockResolvedValue(user) })}
      >
        <StateProbe />
      </AuthSessionProvider>,
    );
    expect(await screen.findByText("bootstrap-error")).toBeInTheDocument();
    fireEvent.click(screen.getByText("retry bootstrap"));
    expect(await screen.findByText("authenticated")).toBeInTheDocument();
    expect(refresh).toHaveBeenCalledTimes(2);
  });

  it("becomes unauthenticated when authoritative /me rejects the refreshed token", async () => {
    render(
      <AuthSessionProvider
        api={fakeApi({
          refresh: vi.fn().mockResolvedValue(access),
          me: vi.fn().mockRejectedValue(invalidAccess()),
        })}
      >
        <StateProbe />
      </AuthSessionProvider>,
    );
    expect(await screen.findByText("unauthenticated")).toBeInTheDocument();
    expect(screen.getByText("anonymous")).toBeInTheDocument();
  });

  it("does not duplicate an in-flight bootstrap under StrictMode", async () => {
    let resolveRefresh: ((value: typeof access) => void) | undefined;
    const refresh = vi.fn(
      () =>
        new Promise<typeof access>((resolve) => {
          resolveRefresh = resolve;
        }),
    );
    render(
      <StrictMode>
        <AuthSessionProvider
          api={fakeApi({ refresh, me: vi.fn().mockResolvedValue(user) })}
        >
          <StateProbe />
        </AuthSessionProvider>
      </StrictMode>,
    );
    expect(refresh).toHaveBeenCalledTimes(1);
    await act(async () => resolveRefresh?.(access));
    expect(await screen.findByText("authenticated")).toBeInTheDocument();
  });
});

describe("AuthSessionProvider recovery and logout", () => {
  it("shares one refresh across simultaneous operations and retries with the new token", async () => {
    const refresh = vi
      .fn()
      .mockRejectedValueOnce(rejectedSession())
      .mockResolvedValueOnce({ ...access, accessToken: "replacement" });
    const api = fakeApi({ refresh });
    const results: string[] = [];
    function ConcurrentProbe() {
      const { establishSession, runAuthenticated } = useAuthSession();
      return (
        <button
          onClick={() => {
            establishSession(sessionResponse);
            const operation = vi.fn(async (token: string) => {
              if (token === "memory-access") throw invalidAccess();
              return token;
            });
            void Promise.all([
              runAuthenticated(operation),
              runAuthenticated(operation),
              runAuthenticated(operation),
            ]).then((values) => results.push(...values));
          }}
        >
          run three
        </button>
      );
    }
    render(
      <AuthSessionProvider api={api} coordinator={fakeCoordinator()}>
        <ConcurrentProbe />
      </AuthSessionProvider>,
    );
    await waitFor(() => expect(refresh).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByText("run three"));
    await waitFor(() =>
      expect(results).toEqual(["replacement", "replacement", "replacement"]),
    );
    expect(refresh).toHaveBeenCalledTimes(2);
  });

  it("never refreshes twice after the retried operation rejects its token", async () => {
    const refresh = vi
      .fn()
      .mockRejectedValueOnce(rejectedSession())
      .mockResolvedValueOnce({ ...access, accessToken: "replacement" });
    let caught = false;
    function RetryProbe() {
      const { state, establishSession, runAuthenticated } = useAuthSession();
      return (
        <>
          <output>{state.status}</output>
          <button
            onClick={() => {
              establishSession(sessionResponse);
              void runAuthenticated(async () => {
                throw invalidAccess();
              }).catch(() => {
                caught = true;
              });
            }}
          >
            retry once
          </button>
        </>
      );
    }
    render(
      <AuthSessionProvider api={fakeApi({ refresh })}>
        <RetryProbe />
      </AuthSessionProvider>,
    );
    await screen.findByText("unauthenticated");
    fireEvent.click(screen.getByText("retry once"));
    await waitFor(() => expect(caught).toBe(true));
    expect(refresh).toHaveBeenCalledTimes(2);
    expect(screen.getByText("unauthenticated")).toBeInTheDocument();
  });

  it("does not refresh non-token failures", async () => {
    const refresh = vi.fn().mockRejectedValue(rejectedSession());
    let caught = false;
    function FailureProbe() {
      const { establishSession, runAuthenticated } = useAuthSession();
      return (
        <button
          onClick={() => {
            establishSession(sessionResponse);
            void runAuthenticated(async () => {
              throw new AuthApiError(503, "AUTH_SERVICE_UNAVAILABLE", "down");
            }).catch(() => {
              caught = true;
            });
          }}
        >
          fail
        </button>
      );
    }
    render(
      <AuthSessionProvider api={fakeApi({ refresh })}>
        <FailureProbe />
      </AuthSessionProvider>,
    );
    await waitFor(() => expect(refresh).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByText("fail"));
    await waitFor(() => expect(caught).toBe(true));
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("retains memory state on a temporary refresh failure and clears the failed flight", async () => {
    const refresh = vi
      .fn()
      .mockRejectedValueOnce(rejectedSession())
      .mockRejectedValueOnce(
        new AuthApiError(503, "AUTH_SERVICE_UNAVAILABLE", "down"),
      )
      .mockResolvedValueOnce({ ...access, accessToken: "replacement" });
    let attempts = 0;
    function RecoveryProbe() {
      const { state, establishSession, runAuthenticated } = useAuthSession();
      const run = () => {
        attempts += 1;
        void runAuthenticated(async (token) => {
          if (token === "memory-access") throw invalidAccess();
          return token;
        }).catch(() => undefined);
      };
      return (
        <>
          <output>{state.status}</output>
          <button onClick={() => establishSession(sessionResponse)}>
            establish recovery
          </button>
          <button onClick={run}>recover</button>
        </>
      );
    }
    render(
      <AuthSessionProvider api={fakeApi({ refresh })}>
        <RecoveryProbe />
      </AuthSessionProvider>,
    );
    await screen.findByText("unauthenticated");
    fireEvent.click(screen.getByText("establish recovery"));
    fireEvent.click(screen.getByText("recover"));
    await waitFor(() => expect(refresh).toHaveBeenCalledTimes(2));
    expect(screen.getByText("authenticated")).toBeInTheDocument();
    fireEvent.click(screen.getByText("recover"));
    await waitFor(() => expect(refresh).toHaveBeenCalledTimes(3));
    expect(attempts).toBe(2);
    expect(screen.getByText("authenticated")).toBeInTheDocument();
  });

  it("clears local state and broadcasts for success, rejection, and service failure", async () => {
    for (const logoutOperation of [
      () => Promise.resolve(undefined),
      () => Promise.reject(rejectedSession()),
      () =>
        Promise.reject(
          new AuthApiError(503, "AUTH_SERVICE_UNAVAILABLE", "down"),
        ),
    ]) {
      const coordinator = fakeCoordinator();
      const { unmount } = render(
        <AuthSessionProvider
          api={fakeApi({ logout: vi.fn(logoutOperation) })}
          coordinator={coordinator}
        >
          <StateProbe />
        </AuthSessionProvider>,
      );
      await screen.findByText("unauthenticated");
      fireEvent.click(screen.getByText("establish"));
      expect(await screen.findByText("person@example.com")).toBeInTheDocument();
      fireEvent.click(screen.getByText("logout"));
      expect(await screen.findByText("anonymous")).toBeInTheDocument();
      expect(coordinator.signals).toContain("logout-complete");
      unmount();
    }
  });

  it("clears memory when another browser context broadcasts invalidation", async () => {
    const coordinator = fakeCoordinator();
    render(
      <AuthSessionProvider api={fakeApi()} coordinator={coordinator}>
        <StateProbe />
      </AuthSessionProvider>,
    );
    await screen.findByText("unauthenticated");
    fireEvent.click(screen.getByText("establish"));
    expect(await screen.findByText("person@example.com")).toBeInTheDocument();
    act(() => coordinator.emit("session-invalidated"));
    expect(await screen.findByText("anonymous")).toBeInTheDocument();
  });
});
