import type { components } from "@portfolio/api-contracts";

import { ApiError } from "@/platform/api/api-error";

export type Portfolio = components["schemas"]["Portfolio"];
export type PortfolioStatus = components["schemas"]["PortfolioStatus"];
export type PortfolioListResponse =
  components["schemas"]["PortfolioListResponse"];
export type CreatePortfolioRequest =
  components["schemas"]["CreatePortfolioRequest"];
export type UpdatePortfolioRequest =
  components["schemas"]["UpdatePortfolioRequest"];
export type ErrorCode = components["schemas"]["ErrorCode"];

const portfolioBasePath = "/api/v1/portfolios";

export class PortfolioApiError extends ApiError {
  constructor(
    status: number,
    code: ErrorCode,
    message: string,
    correlationId?: string,
    retryAfterSeconds?: number,
  ) {
    super(status, code, message, correlationId, retryAfterSeconds);
    this.name = "PortfolioApiError";
  }
}

export interface PortfolioApi {
  list(
    accessToken: string,
    status: PortfolioStatus,
  ): Promise<PortfolioListResponse>;
  get(accessToken: string, portfolioId: string): Promise<Portfolio>;
  create(
    accessToken: string,
    input: CreatePortfolioRequest,
  ): Promise<Portfolio>;
  update(
    accessToken: string,
    portfolioId: string,
    input: UpdatePortfolioRequest,
  ): Promise<Portfolio>;
  archive(accessToken: string, portfolioId: string): Promise<Portfolio>;
}

export const portfolioApi: PortfolioApi = {
  list: (accessToken, status) =>
    requestJSON(
      `?status=${encodeURIComponent(status)}`,
      accessToken,
      {},
      isPortfolioListResponse,
    ),
  get: (accessToken, portfolioId) =>
    requestJSON(
      `/${encodeURIComponent(portfolioId)}`,
      accessToken,
      {},
      isPortfolio,
    ),
  create: (accessToken, input) =>
    requestJSON("", accessToken, jsonRequest("POST", input), isPortfolio),
  update: (accessToken, portfolioId, input) =>
    requestJSON(
      `/${encodeURIComponent(portfolioId)}`,
      accessToken,
      jsonRequest("PATCH", input),
      isPortfolio,
    ),
  archive: (accessToken, portfolioId) =>
    requestJSON(
      `/${encodeURIComponent(portfolioId)}/archive`,
      accessToken,
      { method: "POST" },
      isPortfolio,
    ),
};

async function requestJSON<T>(
  suffix: string,
  accessToken: string,
  init: RequestInit,
  validate: (value: unknown) => value is T,
): Promise<T> {
  const response = await request(suffix, accessToken, init);
  if (!response.ok) throw await parseError(response);
  try {
    const body: unknown = await response.json();
    if (!validate(body)) throw genericError(response);
    return body;
  } catch (error) {
    if (error instanceof PortfolioApiError) throw error;
    throw genericError(response);
  }
}

async function request(
  suffix: string,
  accessToken: string,
  init: RequestInit,
): Promise<Response> {
  try {
    return await fetch(`${portfolioBasePath}${suffix}`, {
      ...init,
      credentials: "omit",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${accessToken}`,
        ...init.headers,
      },
    });
  } catch {
    throw new PortfolioApiError(
      0,
      "INTERNAL_ERROR",
      "Portfolio data is temporarily unavailable. Please try again.",
    );
  }
}

async function parseError(response: Response): Promise<PortfolioApiError> {
  try {
    const body: unknown = await response.json();
    if (isErrorEnvelope(body)) {
      return new PortfolioApiError(
        response.status,
        body.error.code,
        body.error.message,
        body.error.correlationId,
        parseRetryAfter(response.headers.get("Retry-After")),
      );
    }
  } catch {
    // Malformed errors are intentionally collapsed to the safe service error.
  }
  return genericError(response);
}

function genericError(response: Response): PortfolioApiError {
  return new PortfolioApiError(
    response.status,
    "INTERNAL_ERROR",
    "Portfolio data is temporarily unavailable. Please try again.",
    response.headers.get("X-Correlation-ID") ?? undefined,
    parseRetryAfter(response.headers.get("Retry-After")),
  );
}

function jsonRequest(method: "POST" | "PATCH", body: object): RequestInit {
  return {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

function isPortfolio(value: unknown): value is Portfolio {
  if (!isObject(value)) return false;
  const archivedAt = value.archivedAt;
  return (
    typeof value.id === "string" &&
    value.id.length > 0 &&
    typeof value.name === "string" &&
    value.baseCurrency === "USD" &&
    (value.status === "ACTIVE" || value.status === "ARCHIVED") &&
    ((value.status === "ACTIVE" && archivedAt === null) ||
      (value.status === "ARCHIVED" && typeof archivedAt === "string")) &&
    typeof value.createdAt === "string" &&
    typeof value.updatedAt === "string"
  );
}

function isPortfolioListResponse(
  value: unknown,
): value is PortfolioListResponse {
  return (
    isObject(value) &&
    Array.isArray(value.items) &&
    value.items.every(isPortfolio)
  );
}

function isErrorEnvelope(value: unknown): value is {
  error: { code: ErrorCode; message: string; correlationId: string };
} {
  if (!isObject(value) || !isObject(value.error)) return false;
  return (
    typeof value.error.code === "string" &&
    typeof value.error.message === "string" &&
    typeof value.error.correlationId === "string"
  );
}

function isObject(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object";
}

function parseRetryAfter(value: string | null): number | undefined {
  if (!value || !/^\d+$/.test(value)) return undefined;
  const seconds = Number(value);
  return Number.isSafeInteger(seconds) && seconds > 0 ? seconds : undefined;
}
