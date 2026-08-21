import type { components } from "@portfolio/api-contracts";

import { ApiError } from "@/platform/api/api-error";

export type Asset = components["schemas"]["Asset"];
export type AssetType = components["schemas"]["AssetType"];
export type AssetListResponse = components["schemas"]["AssetListResponse"];
export type ErrorCode = components["schemas"]["ErrorCode"];

export type AssetListParams = {
  search?: string;
  assetType?: AssetType;
  cursor?: string;
  limit?: number;
};

const assetBasePath = "/api/v1/assets";

export class AssetApiError extends ApiError {
  constructor(
    status: number,
    code: ErrorCode,
    message: string,
    correlationId?: string,
  ) {
    super(status, code, message, correlationId);
    this.name = "AssetApiError";
  }
}

export interface AssetApi {
  list(
    accessToken: string,
    params: AssetListParams,
  ): Promise<AssetListResponse>;
}

export const assetApi: AssetApi = {
  list: (accessToken, params) =>
    requestJSON(buildListPath(params), accessToken, isAssetListResponse),
};

async function requestJSON(
  path: string,
  accessToken: string,
  validate: (value: unknown) => value is AssetListResponse,
): Promise<AssetListResponse> {
  const response = await request(path, accessToken);
  if (!response.ok) throw await parseError(response);
  try {
    const body: unknown = await response.json();
    if (!validate(body)) throw genericError(response);
    return body;
  } catch (error) {
    if (error instanceof AssetApiError) throw error;
    throw genericError(response);
  }
}

async function request(path: string, accessToken: string): Promise<Response> {
  try {
    return await fetch(path, {
      credentials: "omit",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${accessToken}`,
      },
    });
  } catch {
    throw new AssetApiError(
      0,
      "INTERNAL_ERROR",
      "Asset data is temporarily unavailable. Please try again.",
    );
  }
}

function buildListPath({
  search,
  assetType,
  cursor,
  limit,
}: AssetListParams): string {
  const query = new URLSearchParams();
  if (search !== undefined && search !== "") query.set("search", search);
  if (assetType !== undefined) query.set("type", assetType);
  if (cursor !== undefined) query.set("cursor", cursor);
  if (limit !== undefined) query.set("limit", String(limit));
  const suffix = query.toString();
  return suffix ? `${assetBasePath}?${suffix}` : assetBasePath;
}

async function parseError(response: Response): Promise<AssetApiError> {
  try {
    const body: unknown = await response.json();
    if (isErrorEnvelope(body)) {
      return new AssetApiError(
        response.status,
        body.error.code,
        body.error.message,
        body.error.correlationId,
      );
    }
  } catch {
    // Malformed errors are intentionally collapsed to the safe service error.
  }
  return genericError(response);
}

function genericError(response: Response): AssetApiError {
  return new AssetApiError(
    response.status,
    "INTERNAL_ERROR",
    "Asset data is temporarily unavailable. Please try again.",
    response.headers.get("X-Correlation-ID") ?? undefined,
  );
}

function isAssetListResponse(value: unknown): value is AssetListResponse {
  return (
    isExactRecord(value, ["items", "nextCursor"]) &&
    Array.isArray(value.items) &&
    value.items.every(isAsset) &&
    (value.nextCursor === null ||
      (typeof value.nextCursor === "string" &&
        value.nextCursor.length > 0 &&
        value.nextCursor.length <= 512))
  );
}

function isAsset(value: unknown): value is Asset {
  if (
    !isExactRecord(value, [
      "id",
      "symbol",
      "name",
      "assetType",
      "exchange",
      "currency",
    ])
  ) {
    return false;
  }
  const hasText = (field: unknown) =>
    typeof field === "string" && field.length > 0;
  return (
    hasText(value.id) &&
    hasText(value.symbol) &&
    hasText(value.name) &&
    (value.assetType === "EQUITY" ||
      value.assetType === "ETF" ||
      value.assetType === "CRYPTO") &&
    hasText(value.exchange) &&
    value.currency === "USD" &&
    (value.assetType !== "CRYPTO" || value.exchange === "CRYPTO")
  );
}

function isErrorEnvelope(value: unknown): value is {
  error: { code: ErrorCode; message: string; correlationId: string };
} {
  return (
    isExactRecord(value, ["error"]) &&
    isExactRecord(value.error, ["code", "message", "correlationId"]) &&
    typeof value.error.code === "string" &&
    typeof value.error.message === "string" &&
    typeof value.error.correlationId === "string"
  );
}

function isExactRecord(
  value: unknown,
  keys: readonly string[],
): value is Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  const recordKeys = Object.keys(value).sort();
  const expectedKeys = [...keys].sort();
  return (
    recordKeys.length === expectedKeys.length &&
    recordKeys.every((key, index) => key === expectedKeys[index])
  );
}
