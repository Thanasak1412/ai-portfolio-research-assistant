"use client";

import { useInfiniteQuery } from "@tanstack/react-query";

import { assetApi, type AssetListParams } from "@/features/asset/api/asset-api";
import {
  assetKeys,
  type AssetListFilters,
} from "@/features/asset/model/asset-query-keys";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";

export const defaultAssetPageLimit = 25;

export function useAssets(filters: AssetListFilters) {
  const { runAuthenticated, state } = useAuthSession();
  return useInfiniteQuery({
    queryKey: assetKeys.list(filters),
    enabled: state.status === "authenticated",
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) =>
      runAuthenticated((token) =>
        assetApi.list(token, pageParams(filters, pageParam)),
      ),
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
  });
}

function pageParams(
  filters: AssetListFilters,
  cursor: string | undefined,
): AssetListParams {
  return {
    search: filters.search,
    assetType: filters.assetType,
    limit: filters.limit,
    cursor,
  };
}
