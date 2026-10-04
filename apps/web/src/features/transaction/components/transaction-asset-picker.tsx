"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { Asset } from "@/features/asset/api/asset-api";
import { useAssets } from "@/features/asset/model/asset-queries";
import { assetSearchValidationMessage } from "@/features/asset/model/asset-search-validation";
import { isEligibleAsset } from "@/features/transaction/model/transaction-validation";

export function TransactionAssetPicker({
  selected,
  onSelect,
  validationError,
}: Readonly<{
  selected: Asset | null;
  onSelect: (asset: Asset) => void;
  validationError?: string | null;
}>) {
  const [draft, setDraft] = useState("");
  const [search, setSearch] = useState<string>();
  const [assetType, setAssetType] = useState<"EQUITY" | "ETF">("EQUITY");
  const [error, setError] = useState<string | null>(null);
  const assets = useAssets({ search, assetType, limit: 25 });
  const eligible =
    assets.data?.pages.flatMap((page) => page.items).filter(isEligibleAsset) ??
    [];
  return (
    <fieldset
      className="space-y-3 rounded border p-4"
      aria-invalid={!!validationError}
      aria-describedby={validationError ? "entry-asset-error" : undefined}
    >
      <legend className="font-medium">Canonical Asset</legend>
      {validationError && (
        <p role="alert" id="entry-asset-error">
          {validationError}
        </p>
      )}
      <p className="text-sm">
        Select a USD Equity or ETF on NYSE, NASDAQ, NYSEARCA, or AMEX.
        Eligibility is verified by the server.
      </p>
      <label htmlFor="entry-asset-type">Asset type</label>
      <select
        id="entry-asset-type"
        className="block rounded border p-2"
        value={assetType}
        onChange={(event) =>
          setAssetType(event.target.value as "EQUITY" | "ETF")
        }
      >
        <option value="EQUITY">EQUITY</option>
        <option value="ETF">ETF</option>
      </select>
      <label htmlFor="entry-asset-search">Search Assets</label>
      <Input
        id="entry-asset-search"
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        aria-invalid={!!error}
        aria-describedby={error ? "entry-asset-search-error" : undefined}
      />
      <Button
        type="button"
        variant="outline"
        onClick={() => {
          const message = assetSearchValidationMessage(draft);
          setError(message);
          if (!message) setSearch(draft || undefined);
        }}
      >
        Search
      </Button>
      {error && (
        <p role="alert" id="entry-asset-search-error">
          {error}
        </p>
      )}
      {selected && (
        <p role="status">
          Selected: {selected.symbol} — {selected.name} · {selected.assetType} ·{" "}
          {selected.exchange} · {selected.currency}
        </p>
      )}
      {assets.isLoading && <p role="status">Loading eligible Assets…</p>}
      {assets.isError && (
        <div role="alert">
          Assets could not be loaded.{" "}
          <Button
            type="button"
            variant="outline"
            onClick={() => void assets.refetch()}
          >
            Retry Assets
          </Button>
        </div>
      )}
      {assets.isSuccess && eligible.length === 0 && (
        <p>
          No eligible Assets in these results. Try another search or load more.
        </p>
      )}
      <ul className="space-y-2">
        {eligible.map((asset) => (
          <li key={asset.id}>
            <Button
              type="button"
              variant={selected?.id === asset.id ? "default" : "outline"}
              className="h-auto whitespace-normal text-left"
              aria-pressed={selected?.id === asset.id}
              onClick={() => onSelect(asset)}
            >
              {asset.symbol} — {asset.name} · {asset.assetType} ·{" "}
              {asset.exchange} · {asset.currency}
            </Button>
          </li>
        ))}
      </ul>
      {assets.hasNextPage && (
        <Button
          type="button"
          variant="outline"
          disabled={assets.isFetchingNextPage}
          onClick={() => void assets.fetchNextPage()}
        >
          {assets.isFetchingNextPage ? "Loading Assets…" : "Load more Assets"}
        </Button>
      )}
    </fieldset>
  );
}
