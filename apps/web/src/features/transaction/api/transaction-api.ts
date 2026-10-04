import type { components, operations } from "@portfolio/api-contracts";
import { z } from "zod";

import { transactionSchema } from "@/features/transaction/model/transaction-validation";
import { ApiError } from "@/platform/api/api-error";

export type Transaction = components["schemas"]["Transaction"];
export type TransactionCommand = components["schemas"]["TransactionCommand"];
export type TransactionListResponse =
  components["schemas"]["TransactionListResponse"];
export type TransactionListParams = NonNullable<
  operations["listTransactions"]["parameters"]["query"]
>;
export type TransactionFilters = Omit<TransactionListParams, "cursor">;

const listSchema = z.strictObject({
  items: z.array(transactionSchema),
  nextCursor: z.string().min(1).max(512).nullable(),
});
const errorSchema = z.strictObject({
  error: z.strictObject({
    code: z.string(),
    message: z.string(),
    correlationId: z.string(),
  }),
});
const unavailable = "Transaction service is unavailable. Please try again.";
const path = (id: string) =>
  `/api/v1/portfolios/${encodeURIComponent(id)}/transactions`;

export const transactionApi = {
  list(
    accessToken: string,
    portfolioId: string,
    params: TransactionListParams,
  ): Promise<TransactionListResponse> {
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(params))
      if (value !== undefined) query.set(key, String(value));
    return request(
      `${path(portfolioId)}?${query}`,
      accessToken,
      {},
      200,
      listSchema,
    );
  },
  create(
    accessToken: string,
    portfolioId: string,
    command: TransactionCommand,
    idempotencyKey: string,
  ): Promise<Transaction> {
    return request(
      path(portfolioId),
      accessToken,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": idempotencyKey,
        },
        body: JSON.stringify(command),
      },
      201,
      transactionSchema,
    );
  },
};

async function request<T>(
  url: string,
  token: string,
  init: RequestInit,
  expectedStatus: number,
  schema: z.ZodType<T>,
): Promise<T> {
  let response: Response;
  try {
    response = await fetch(url, {
      ...init,
      credentials: "omit",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
        ...init.headers,
      },
    });
  } catch {
    throw new ApiError(0, "INTERNAL_ERROR", unavailable);
  }
  const generic = () =>
    new ApiError(
      response.status,
      "INTERNAL_ERROR",
      unavailable,
      response.headers.get("X-Correlation-ID") ?? undefined,
    );
  let body: unknown;
  try {
    body = await response.json();
  } catch {
    throw generic();
  }
  if (!response.ok) {
    const parsed = errorSchema.safeParse(body);
    if (!parsed.success) throw generic();
    // Raw backend messages are not presented as UI instructions or validation.
    throw new ApiError(
      response.status,
      parsed.data.error.code,
      unavailable,
      parsed.data.error.correlationId,
    );
  }
  const parsed = schema.safeParse(body);
  if (response.status !== expectedStatus || !parsed.success) throw generic();
  return parsed.data;
}
