import { Button } from "@/components/ui/button";

export function AssetEmptyState({
  filtered,
  onClearFilters,
}: Readonly<{
  filtered: boolean;
  onClearFilters: () => void;
}>) {
  return (
    <section className="rounded-lg border border-dashed border-slate-300 p-6 text-sm text-slate-600">
      <p>
        {filtered
          ? "No assets match the current search and filter."
          : "No supported assets are available."}
      </p>
      {filtered ? (
        <Button
          type="button"
          variant="outline"
          className="mt-3"
          onClick={onClearFilters}
        >
          Clear filters
        </Button>
      ) : null}
    </section>
  );
}
