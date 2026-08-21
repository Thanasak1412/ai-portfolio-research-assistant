"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import type { AssetType } from "@/features/asset/api/asset-api";
import { AssetEmptyState } from "@/features/asset/components/asset-empty-state";
import { AssetError } from "@/features/asset/components/asset-error";
import { AssetList } from "@/features/asset/components/asset-list";
import { AssetSearchForm } from "@/features/asset/components/asset-search-form";
import { AssetTypeFilter } from "@/features/asset/components/asset-type-filter";
import {
  defaultAssetPageLimit,
  useAssets,
} from "@/features/asset/model/asset-queries";
import { assetSearchValidationMessage } from "@/features/asset/model/asset-search-validation";

export function AssetDiscoveryScreen() {
  const [draftSearch, setDraftSearch] = useState("");
  const [search, setSearch] = useState<string | undefined>();
  const [assetType, setAssetType] = useState<AssetType | undefined>();
  const [searchError, setSearchError] = useState<string | null>(null);
  const assets = useAssets({ search, assetType, limit: defaultAssetPageLimit });
  const filtered = search !== undefined || assetType !== undefined;
  const assetItems = assets.data?.pages.flatMap((page) => page.items) ?? [];

  function applySearch() {
    const validationError = assetSearchValidationMessage(draftSearch);
    if (validationError) {
      setSearchError(validationError);
      return;
    }
    setSearchError(null);
    setSearch(draftSearch === "" ? undefined : draftSearch);
  }

  function clearSearch() {
    setDraftSearch("");
    setSearchError(null);
    setSearch(undefined);
  }

  function clearFilters() {
    clearSearch();
    setAssetType(undefined);
  }

  return (
    <main className="mx-auto max-w-3xl space-y-6 px-4 py-8 sm:px-6">
      <header>
        <h1 className="text-2xl font-semibold">Assets</h1>
        <p className="mt-1 text-sm text-slate-600">
          Browse system-managed canonical Asset reference metadata.
        </p>
        <p className="mt-2 text-sm text-slate-600">
          Prices, holdings, transactions, and valuation are not part of this
          view.
        </p>
      </header>
      <AssetSearchForm
        value={draftSearch}
        error={searchError}
        onChange={setDraftSearch}
        onSubmit={applySearch}
        onClear={clearSearch}
      />
      <AssetTypeFilter selected={assetType} onSelect={setAssetType} />
      {assets.isLoading ? <p role="status">Loading assets…</p> : null}
      {assets.isError ? (
        <div className="space-y-3">
          <AssetError error={assets.error} />
          <Button
            type="button"
            variant="outline"
            onClick={() => void assets.refetch()}
          >
            Retry
          </Button>
        </div>
      ) : null}
      {assets.isSuccess && assetItems.length === 0 ? (
        <AssetEmptyState filtered={filtered} onClearFilters={clearFilters} />
      ) : null}
      {assets.isSuccess && assetItems.length > 0 ? (
        <AssetList assets={assetItems} />
      ) : null}
      {assets.hasNextPage ? (
        <Button
          type="button"
          variant="outline"
          disabled={assets.isFetchingNextPage}
          onClick={() => void assets.fetchNextPage()}
        >
          {assets.isFetchingNextPage ? "Loading more…" : "Load more"}
        </Button>
      ) : null}
    </main>
  );
}
