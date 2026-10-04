import type { TransactionFilters } from "@/features/transaction/api/transaction-api";

export const transactionKeys = {
  all: ["transactions"] as const,
  portfolio: (id: string) => [...transactionKeys.all, id] as const,
  histories: (id: string) =>
    [...transactionKeys.portfolio(id), "history"] as const,
  details: (id: string) =>
    [...transactionKeys.portfolio(id), "detail"] as const,
  detail: (id: string, transactionId: string) =>
    [...transactionKeys.details(id), transactionId] as const,
  history: (id: string, filters: TransactionFilters) =>
    [
      ...transactionKeys.histories(id),
      {
        kind: filters.kind ?? null,
        effectiveAtFrom: filters.effectiveAtFrom ?? null,
        effectiveAtTo: filters.effectiveAtTo ?? null,
        includeReversals: filters.includeReversals ?? true,
        limit: filters.limit ?? 50,
      },
    ] as const,
};
