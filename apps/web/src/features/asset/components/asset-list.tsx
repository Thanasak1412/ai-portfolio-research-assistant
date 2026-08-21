import type { Asset } from "@/features/asset/api/asset-api";

export function AssetList({ assets }: Readonly<{ assets: Asset[] }>) {
  return (
    <ul className="grid gap-3 sm:grid-cols-2" aria-label="Assets">
      {assets.map((asset) => (
        <li
          key={asset.id}
          className="min-w-0 rounded-lg border border-slate-200 bg-white p-4 shadow-sm"
        >
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div className="min-w-0">
              <h2 className="break-words font-semibold">{asset.symbol}</h2>
              <p className="break-words text-sm text-slate-600">{asset.name}</p>
            </div>
            <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium">
              {asset.assetType}
            </span>
          </div>
          <dl className="mt-3 grid grid-cols-2 gap-2 text-sm">
            <div>
              <dt className="text-slate-500">Exchange</dt>
              <dd className="break-words font-medium">{asset.exchange}</dd>
            </div>
            <div>
              <dt className="text-slate-500">Currency</dt>
              <dd className="font-medium">{asset.currency}</dd>
            </div>
          </dl>
        </li>
      ))}
    </ul>
  );
}
