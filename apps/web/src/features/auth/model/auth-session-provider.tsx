"use client";

import { createContext, useContext, useMemo, useState } from "react";

import type {
  AccessTokenResponse,
  AuthenticatedSessionResponse,
} from "@/features/auth/api/auth-api";
import {
  replaceSessionAccessToken,
  sessionFromResponse,
  type AuthenticatedSession,
} from "@/features/auth/model/auth-session";

type AuthSessionContextValue = {
  session: AuthenticatedSession | null;
  establishSession: (response: AuthenticatedSessionResponse) => void;
  replaceAccessToken: (response: AccessTokenResponse) => void;
  clearMemorySession: () => void;
};

const AuthSessionContext = createContext<AuthSessionContextValue | null>(null);

export function AuthSessionProvider({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const [session, setSession] = useState<AuthenticatedSession | null>(null);
  const value = useMemo<AuthSessionContextValue>(
    () => ({
      session,
      establishSession: (response) => setSession(sessionFromResponse(response)),
      replaceAccessToken: (response) =>
        setSession((current) =>
          current ? replaceSessionAccessToken(current, response) : current,
        ),
      clearMemorySession: () => setSession(null),
    }),
    [session],
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
