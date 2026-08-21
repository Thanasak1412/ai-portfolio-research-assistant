import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function AssetSearchForm({
  value,
  error,
  onChange,
  onSubmit,
  onClear,
}: Readonly<{
  value: string;
  error: string | null;
  onChange: (next: string) => void;
  onSubmit: () => void;
  onClear: () => void;
}>) {
  return (
    <form
      className="flex flex-col gap-3 sm:flex-row sm:items-start"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
      noValidate
    >
      <div className="min-w-0 flex-1">
        <label htmlFor="asset-search" className="text-sm font-medium">
          Search assets
        </label>
        <Input
          id="asset-search"
          value={value}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={!!error}
          aria-describedby={error ? "asset-search-error" : undefined}
        />
        {error ? (
          <p
            id="asset-search-error"
            role="alert"
            className="mt-1 text-sm text-red-700"
          >
            {error}
          </p>
        ) : null}
      </div>
      <div className="flex gap-2 sm:mt-6">
        <Button type="submit">Search</Button>
        <Button type="button" variant="outline" onClick={onClear}>
          Clear search
        </Button>
      </div>
    </form>
  );
}
