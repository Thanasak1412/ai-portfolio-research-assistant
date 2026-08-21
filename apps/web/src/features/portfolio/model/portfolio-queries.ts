"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  type CreatePortfolioRequest,
  portfolioApi,
  type Portfolio,
  type PortfolioStatus,
  type UpdatePortfolioRequest,
} from "@/features/portfolio/api/portfolio-api";
import { portfolioKeys } from "@/features/portfolio/model/portfolio-query-keys";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";

export function usePortfolios(status: PortfolioStatus) {
  const { runAuthenticated, state } = useAuthSession();
  return useQuery({
    queryKey: portfolioKeys.list(status),
    enabled: state.status === "authenticated",
    queryFn: () =>
      runAuthenticated((token) => portfolioApi.list(token, status)),
  });
}

export function usePortfolio(portfolioId: string) {
  const { runAuthenticated, state } = useAuthSession();
  return useQuery({
    queryKey: portfolioKeys.detail(portfolioId),
    enabled: state.status === "authenticated" && portfolioId.length > 0,
    queryFn: () =>
      runAuthenticated((token) => portfolioApi.get(token, portfolioId)),
  });
}

export function useCreatePortfolio() {
  const { runAuthenticated } = useAuthSession();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreatePortfolioRequest) =>
      runAuthenticated((token) => portfolioApi.create(token, input)),
    onSuccess: async (portfolio) => {
      queryClient.setQueryData(portfolioKeys.detail(portfolio.id), portfolio);
      await queryClient.invalidateQueries({
        queryKey: portfolioKeys.list("ACTIVE"),
      });
    },
  });
}

export function useUpdatePortfolio() {
  const { runAuthenticated } = useAuthSession();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      portfolioId,
      input,
    }: {
      portfolioId: string;
      input: UpdatePortfolioRequest;
    }) =>
      runAuthenticated((token) =>
        portfolioApi.update(token, portfolioId, input),
      ),
    onSuccess: async (portfolio) => {
      cachePortfolio(queryClient, portfolio);
      await queryClient.invalidateQueries({ queryKey: portfolioKeys.lists() });
    },
  });
}

export function useArchivePortfolio() {
  const { runAuthenticated } = useAuthSession();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (portfolioId: string) =>
      runAuthenticated((token) => portfolioApi.archive(token, portfolioId)),
    onSuccess: async (portfolio) => {
      cachePortfolio(queryClient, portfolio);
      await queryClient.invalidateQueries({
        queryKey: portfolioKeys.list("ACTIVE"),
      });
      await queryClient.invalidateQueries({
        queryKey: portfolioKeys.list("ARCHIVED"),
      });
    },
  });
}

function cachePortfolio(
  queryClient: ReturnType<typeof useQueryClient>,
  portfolio: Portfolio,
) {
  queryClient.setQueryData(portfolioKeys.detail(portfolio.id), portfolio);
}
