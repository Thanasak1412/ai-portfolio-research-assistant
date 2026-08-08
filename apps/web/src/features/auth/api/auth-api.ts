import type { components } from "@portfolio/api-contracts";

export type CredentialsRequest = components["schemas"]["CredentialsRequest"];
export type AuthenticatedUser = components["schemas"]["AuthenticatedUser"];
export type AuthenticatedSessionResponse =
  components["schemas"]["AuthenticatedSessionResponse"];
export type AccessTokenResponse = components["schemas"]["AccessTokenResponse"];
export type ErrorCode = components["schemas"]["ErrorCode"];

const authBasePath = "/api/v1/auth";

export class AuthApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: ErrorCode,
    message: string,
    readonly correlationId?: string,
    readonly retryAfterSeconds?: number,
  ) {
    super(message);
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
    requestJSON("/refresh", {
      method: "POST",
      headers: requestedWithHeaders(),
    }),
  logout: () =>
    requestVoid("/logout", { method: "POST", headers: requestedWithHeaders() }),
  me: (accessToken) =>
    requestJSON("/me", { headers: { Authorization: `Bearer ${accessToken}` } }),
};

async function requestSession(
  path: string,
  input: CredentialsRequest,
): Promise<AuthenticatedSessionResponse> {
  return requestJSON(path, {
    method: "POST",
    headers: jsonHeaders(),
    body: JSON.stringify(input),
  });
}

async function requestJSON<T>(path: string, init: RequestInit): Promise<T> {
  const response = await request(path, init);
  if (!response.ok) throw await parseError(response);
  try {
    return (await response.json()) as T;
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
