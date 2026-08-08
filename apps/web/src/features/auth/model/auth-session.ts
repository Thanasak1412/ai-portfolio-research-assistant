import type {
  AccessTokenResponse,
  AuthenticatedSessionResponse,
  AuthenticatedUser,
} from "@/features/auth/api/auth-api";

export type AuthenticatedSession = {
  accessToken: string;
  user: AuthenticatedUser;
  accessTokenExpiresAt: number;
};

export function sessionFromResponse(
  response: AuthenticatedSessionResponse,
): AuthenticatedSession {
  return {
    accessToken: response.accessToken,
    user: response.user,
    accessTokenExpiresAt: Date.now() + response.expiresIn * 1_000,
  };
}

export function replaceSessionAccessToken(
  session: AuthenticatedSession,
  response: AccessTokenResponse,
): AuthenticatedSession {
  return {
    ...session,
    accessToken: response.accessToken,
    accessTokenExpiresAt: Date.now() + response.expiresIn * 1_000,
  };
}
