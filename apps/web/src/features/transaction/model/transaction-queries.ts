"use client";

import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";
import {
  transactionApi,
  type TransactionCommand,
  type TransactionFilters,
  type TransactionCorrectionCommand,
} from "@/features/transaction/api/transaction-api";
import { transactionKeys } from "@/features/transaction/model/transaction-query-keys";
import { ApiError } from "@/platform/api/api-error";

export function useTransaction(portfolioId: string, transactionId: string) {
  const { runAuthenticated, state } = useAuthSession();
  return useQuery({
    queryKey: transactionKeys.detail(portfolioId, transactionId),
    enabled:
      state.status === "authenticated" && !!portfolioId && !!transactionId,
    queryFn: () =>
      runAuthenticated((token) =>
        transactionApi.get(token, portfolioId, transactionId),
      ),
  });
}

export function useCorrectTransaction(portfolioId: string) {
  const { runAuthenticated } = useAuthSession();
  const client = useQueryClient();
  const refresh = () => {
    void client.invalidateQueries({
      queryKey: transactionKeys.histories(portfolioId),
    });
    void client.invalidateQueries({
      queryKey: transactionKeys.details(portfolioId),
    });
  };
  return useMutation({
    retry: false,
    gcTime: 0,
    mutationFn: ({
      transactionId,
      command,
      key,
    }: {
      transactionId: string;
      command: TransactionCorrectionCommand;
      key: string;
    }) =>
      runAuthenticated((token) =>
        transactionApi.correct(token, portfolioId, transactionId, command, key),
      ),
    onSuccess: refresh,
    onError: (error) => {
      if (
        error instanceof ApiError &&
        error.code === "TRANSACTION_ALREADY_CORRECTED"
      )
        refresh();
    },
  });
}

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
