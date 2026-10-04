"use client";

import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";
import {
  transactionApi,
  type TransactionCommand,
  type TransactionFilters,
} from "@/features/transaction/api/transaction-api";
import { transactionKeys } from "@/features/transaction/model/transaction-query-keys";

export function useTransactionHistory(
  portfolioId: string,
  filters: TransactionFilters,
) {
  const { runAuthenticated, state } = useAuthSession();
  return useInfiniteQuery({
    queryKey: transactionKeys.history(portfolioId, filters),
    enabled: state.status === "authenticated" && !!portfolioId,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) =>
      runAuthenticated((token) =>
        transactionApi.list(token, portfolioId, {
          ...filters,
          limit: filters.limit ?? 50,
          cursor: pageParam,
        }),
      ),
    getNextPageParam: (page) => page.nextCursor ?? undefined,
  });
}

export function useCreateTransaction(portfolioId: string) {
  const { runAuthenticated } = useAuthSession();
  const queryClient = useQueryClient();
  return useMutation({
    retry: false,
    gcTime: 0,
    mutationFn: ({
      command,
      key,
    }: {
      command: TransactionCommand;
      key: string;
    }) =>
      runAuthenticated((token) =>
        transactionApi.create(token, portfolioId, command, key),
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: transactionKeys.histories(portfolioId),
      });
    },
  });
}
