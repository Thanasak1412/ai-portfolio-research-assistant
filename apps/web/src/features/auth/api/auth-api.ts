import type { components } from "@portfolio/api-contracts";

import { ApiError } from "@/platform/api/api-error";

export type CredentialsRequest = components["schemas"]["CredentialsRequest"];
export type AuthenticatedUser = components["schemas"]["AuthenticatedUser"];
export type AuthenticatedSessionResponse =
  components["schemas"]["AuthenticatedSessionResponse"];
export type AccessTokenResponse = components["schemas"]["AccessTokenResponse"];
export type ErrorCode = components["schemas"]["ErrorCode"];

const authBasePath = "/api/v1/auth";

export class AuthApiError extends ApiError {
  constructor(
    status: number,
    code: ErrorCode,
    message: string,
    correlationId?: string,
    retryAfterSeconds?: number,
  ) {
    super(status, code, message, correlationId, retryAfterSeconds);
    this.name = "AuthApiError";
  }
}

export interface AuthApi {
  register(input: CredentialsRequest): Promise<AuthenticatedSessionResponse>;
  login(input: CredentialsRequest): Promise<AuthenticatedSessionResponse>;
  refresh(): Promise<AccessTokenResponse>;
  logout(): Promise<void>;
  me(accessToken: string): Promise<AuthenticatedUser>;
}

export const authApi: AuthApi = {
  register: (input) => requestSession("/register", input),
  login: (input) => requestSession("/login", input),
  refresh: () =>
    requestJSON(
      "/refresh",
      {
        method: "POST",
        headers: requestedWithHeaders(),
      },
      isAccessTokenResponse,
    ),
  logout: () =>
    requestVoid("/logout", { method: "POST", headers: requestedWithHeaders() }),
  me: (accessToken) =>
    requestJSON(
      "/me",
      { headers: { Authorization: `Bearer ${accessToken}` } },
      isAuthenticatedUser,
    ),
};

async function requestSession(
  path: string,
  input: CredentialsRequest,
): Promise<AuthenticatedSessionResponse> {
  return requestJSON(
    path,
    {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify(input),
    },
    isAuthenticatedSessionResponse,
  );
}

async function requestJSON<T>(
  path: string,
  init: RequestInit,
  validate: (value: unknown) => value is T,
): Promise<T> {
  const response = await request(path, init);
  if (!response.ok) throw await parseError(response);
  try {
    const body: unknown = await response.json();
    if (!validate(body)) throw genericError(response);
    return body;
  } catch {
    throw genericError(response);
  }
}

async function requestVoid(path: string, init: RequestInit): Promise<void> {
  const response = await request(path, init);
  if (!response.ok) throw await parseError(response);
}

async function request(path: string, init: RequestInit): Promise<Response> {
  try {
    return await fetch(`${authBasePath}${path}`, {
      ...init,
      credentials: "same-origin",
      headers: { Accept: "application/json", ...init.headers },
    });
  } catch {
    throw new AuthApiError(
      0,
      "AUTH_SERVICE_UNAVAILABLE",
      "Authentication is temporarily unavailable",
    );
  }
}

async function parseError(response: Response): Promise<AuthApiError> {
  let body: unknown;
  try {
    body = await response.json();
  } catch {
    return genericError(response);
  }
  if (!isErrorEnvelope(body)) return genericError(response);
  const retryAfterSeconds = parseRetryAfter(
    response.headers.get("Retry-After"),
  );
  return new AuthApiError(
    response.status,
    body.error.code,
    body.error.message,
    body.error.correlationId,
    retryAfterSeconds,
  );
}

function genericError(response: Response): AuthApiError {
  return new AuthApiError(
    response.status,
    "INTERNAL_ERROR",
    "Authentication is temporarily unavailable",
    response.headers.get("X-Correlation-ID") ?? undefined,
    parseRetryAfter(response.headers.get("Retry-After")),
  );
}

function isErrorEnvelope(value: unknown): value is {
  error: { code: ErrorCode; message: string; correlationId: string };
} {
  if (!value || typeof value !== "object" || !("error" in value)) return false;
  const error = value.error;
  return (
    !!error &&
    typeof error === "object" &&
    "code" in error &&
    "message" in error &&
    "correlationId" in error &&
    typeof error.code === "string" &&
    typeof error.message === "string" &&
    typeof error.correlationId === "string"
  );
}

function parseRetryAfter(value: string | null): number | undefined {
  if (!value || !/^\d+$/.test(value)) return undefined;
  const seconds = Number(value);
  return Number.isSafeInteger(seconds) && seconds > 0 ? seconds : undefined;
}

function jsonHeaders() {
  return { "Content-Type": "application/json" };
}
function requestedWithHeaders() {
  return { "X-Requested-With": "portfolio-web" };
}

function isAccessTokenResponse(value: unknown): value is AccessTokenResponse {
  if (!isObject(value)) return false;
  return (
    typeof value.accessToken === "string" &&
    value.accessToken.length > 0 &&
    value.tokenType === "Bearer" &&
    typeof value.expiresIn === "number" &&
    Number.isSafeInteger(value.expiresIn) &&
    value.expiresIn > 0
  );
}

function isAuthenticatedSessionResponse(
  value: unknown,
): value is AuthenticatedSessionResponse {
  return (
    isAccessTokenResponse(value) &&
    isAuthenticatedUser((value as Record<string, unknown>).user)
  );
}

function isAuthenticatedUser(value: unknown): value is AuthenticatedUser {
  if (!isObject(value)) return false;
  return (
    typeof value.id === "string" &&
    value.id.length > 0 &&
    typeof value.email === "string" &&
    value.email.length > 0 &&
    (value.status === "active" || value.status === "disabled") &&
    typeof value.createdAt === "string" &&
    typeof value.updatedAt === "string"
  );
}

function isObject(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object";
}
