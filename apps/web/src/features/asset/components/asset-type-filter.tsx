import { Button } from "@/components/ui/button";
import type { AssetType } from "@/features/asset/api/asset-api";

const filters: ReadonlyArray<{ label: string; value?: AssetType }> = [
  { label: "All" },
  { label: "Equity", value: "EQUITY" },
  { label: "ETF", value: "ETF" },
  { label: "Crypto", value: "CRYPTO" },
];

export function AssetTypeFilter({
  selected,
  onSelect,
}: Readonly<{
  selected?: AssetType;
  onSelect: (next?: AssetType) => void;
}>) {
  return (
    <div className="flex flex-wrap gap-2" aria-label="Asset type">
      {filters.map((filter) => (
        <Button
          key={filter.label}
          type="button"
          variant={selected === filter.value ? "default" : "outline"}
          aria-pressed={selected === filter.value}
          onClick={() => onSelect(filter.value)}
        >
          {filter.label}
        </Button>
      ))}
    </div>
  );
}
