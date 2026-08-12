"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  type AccessTokenResponse,
  type AuthApi,
  AuthApiError,
  authApi,
  type AuthenticatedSessionResponse,
} from "@/features/auth/api/auth-api";
import {
  type BrowserSessionCoordinator,
  createBrowserSessionCoordinator,
} from "@/features/auth/model/auth-browser-coordinator";
import {
  replaceSessionAccessToken,
  sessionFromBootstrap,
  sessionFromResponse,
  type AuthenticatedSession,
} from "@/features/auth/model/auth-session";

export type AuthState =
  | { status: "bootstrapping"; session: null }
  | { status: "bootstrap-error"; session: null }
  | { status: "unauthenticated"; session: null }
  | { status: "authenticated"; session: AuthenticatedSession };

export type LogoutResult = { serverRevocationConfirmed: boolean };

type AuthSessionContextValue = {
  state: AuthState;
  session: AuthenticatedSession | null;
  establishSession: (response: AuthenticatedSessionResponse) => void;
  retryBootstrap: () => void;
  runAuthenticated: <T>(
    operation: (accessToken: string) => Promise<T>,
  ) => Promise<T>;
  logout: () => Promise<LogoutResult>;
};

type AuthSessionProviderProps = {
  children: React.ReactNode;
  api?: AuthApi;
  coordinator?: BrowserSessionCoordinator;
};

const initialState: AuthState = { status: "bootstrapping", session: null };
const AuthSessionContext = createContext<AuthSessionContextValue | null>(null);

export function AuthSessionProvider({
  children,
  api = authApi,
  coordinator,
}: Readonly<AuthSessionProviderProps>) {
  const [state, setState] = useState<AuthState>(initialState);
  const [bootstrapAttempt, setBootstrapAttempt] = useState(0);
  const stateRef = useRef<AuthState>(initialState);
  const refreshPromiseRef = useRef<Promise<AccessTokenResponse> | null>(null);
  const bootstrapPromiseRef = useRef<Promise<void> | null>(null);
  const [browserCoordinator] = useState(
    () => coordinator ?? createBrowserSessionCoordinator(),
  );

  const updateState = useCallback((next: AuthState) => {
    stateRef.current = next;
    setState(next);
  }, []);

  const invalidateSession = useCallback(
    (broadcast = true) => {
      updateState({ status: "unauthenticated", session: null });
      if (broadcast) browserCoordinator.broadcast("session-invalidated");
    },
    [browserCoordinator, updateState],
  );

  const refreshAccessToken = useCallback(
    (broadcastRejection = true): Promise<AccessTokenResponse> => {
      if (refreshPromiseRef.current) return refreshPromiseRef.current;
      const refreshPromise = browserCoordinator
        .withSessionTransition(() => api.refresh())
        .then((access) => {
          const current = stateRef.current;
          if (current.status === "authenticated") {
            updateState({
              status: "authenticated",
              session: replaceSessionAccessToken(current.session, access),
            });
          }
          return access;
        })
        .catch((error: unknown) => {
          if (isSessionRefreshRejected(error))
            invalidateSession(broadcastRejection);
          throw error;
        })
        .finally(() => {
          refreshPromiseRef.current = null;
        });
      refreshPromiseRef.current = refreshPromise;
      return refreshPromise;
    },
    [api, browserCoordinator, invalidateSession, updateState],
  );

  const bootstrap = useCallback((): Promise<void> => {
    if (bootstrapPromiseRef.current) return bootstrapPromiseRef.current;
    updateState({ status: "bootstrapping", session: null });
    const bootstrapPromise = refreshAccessToken(false)
      .then(async (access) => {
        const user = await api.me(access.accessToken);
        updateState({
          status: "authenticated",
          session: sessionFromBootstrap(access, user),
        });
      })
      .catch((error: unknown) => {
        if (isSessionRefreshRejected(error) || isInvalidAccessToken(error)) {
          invalidateSession(false);
          return;
        }
        updateState({ status: "bootstrap-error", session: null });
      })
      .finally(() => {
        bootstrapPromiseRef.current = null;
      });
    bootstrapPromiseRef.current = bootstrapPromise;
    return bootstrapPromise;
  }, [api, invalidateSession, refreshAccessToken, updateState]);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap, bootstrapAttempt]);

  useEffect(
    () =>
      browserCoordinator.subscribe((signal) => {
        if (signal === "session-invalidated" || signal === "logout-complete") {
          invalidateSession(false);
        }
      }),
    [browserCoordinator, invalidateSession],
  );

  const establishSession = useCallback(
    (response: AuthenticatedSessionResponse) => {
      updateState({
        status: "authenticated",
        session: sessionFromResponse(response),
      });
      browserCoordinator.broadcast("session-established");
    },
    [browserCoordinator, updateState],
  );

  const runAuthenticated = useCallback(
    async <T,>(operation: (accessToken: string) => Promise<T>): Promise<T> => {
      const current = stateRef.current;
      if (current.status !== "authenticated") throw invalidAccessTokenError();
      const attemptedAccessToken = current.session.accessToken;
      let invalidTokenError: AuthApiError;
      try {
        return await operation(attemptedAccessToken);
      } catch (error) {
        if (!isInvalidAccessToken(error)) throw error;
        invalidTokenError = error;
      }
      const latest = stateRef.current;
      if (latest.status !== "authenticated") throw invalidTokenError;
      const retryAccessToken =
        latest.session.accessToken !== attemptedAccessToken
          ? latest.session.accessToken
          : (await refreshAccessToken()).accessToken;
      try {
        return await operation(retryAccessToken);
      } catch (error) {
        if (isInvalidAccessToken(error)) invalidateSession();
        throw error;
      }
    },
    [invalidateSession, refreshAccessToken],
  );

  const logout = useCallback(async (): Promise<LogoutResult> => {
    let serverRevocationConfirmed = false;
    try {
      await browserCoordinator.withSessionTransition(() => api.logout());
      serverRevocationConfirmed = true;
    } catch (error) {
      if (isSessionRefreshRejected(error)) serverRevocationConfirmed = true;
    } finally {
      updateState({ status: "unauthenticated", session: null });
      browserCoordinator.broadcast("logout-complete");
    }
    return { serverRevocationConfirmed };
  }, [api, browserCoordinator, updateState]);

  const value = useMemo<AuthSessionContextValue>(
    () => ({
      state,
      session: state.session,
      establishSession,
      retryBootstrap: () => setBootstrapAttempt((attempt) => attempt + 1),
      runAuthenticated,
      logout,
    }),
    [establishSession, logout, runAuthenticated, state],
  );

  return (
    <AuthSessionContext.Provider value={value}>
      {children}
    </AuthSessionContext.Provider>
  );
}

export function useAuthSession(): AuthSessionContextValue {
  const value = useContext(AuthSessionContext);
  if (!value)
    throw new Error("useAuthSession must be used inside AuthSessionProvider");
  return value;
}

function isInvalidAccessToken(error: unknown): error is AuthApiError {
  return (
    error instanceof AuthApiError &&
    error.status === 401 &&
    error.code === "ACCESS_TOKEN_INVALID"
  );
}

function isSessionRefreshRejected(error: unknown): error is AuthApiError {
  return (
    error instanceof AuthApiError && error.code === "SESSION_REFRESH_REJECTED"
  );
}

function invalidAccessTokenError(): AuthApiError {
  return new AuthApiError(
    401,
    "ACCESS_TOKEN_INVALID",
    "Authentication required",
  );
}
