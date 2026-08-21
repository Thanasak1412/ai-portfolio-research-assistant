import type { AssetType } from "@/features/asset/api/asset-api";

export type AssetListFilters = {
  search?: string;
  assetType?: AssetType;
  limit: number;
};

export const assetKeys = {
  all: ["assets"] as const,
  lists: () => [...assetKeys.all, "list"] as const,
  list: ({ search, assetType, limit }: AssetListFilters) =>
    [
      ...assetKeys.lists(),
      { search: search ?? null, assetType: assetType ?? null, limit },
    ] as const,
};
