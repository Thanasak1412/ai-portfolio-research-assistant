import Link from "next/link";

import type { Portfolio } from "@/features/portfolio/api/portfolio-api";

export function PortfolioList({
  portfolios,
}: Readonly<{ portfolios: Portfolio[] }>) {
  return (
    <ul className="space-y-3" aria-label="Portfolios">
      {portfolios.map((portfolio) => (
        <li key={portfolio.id}>
          <Link
            href={`/app/portfolios/${encodeURIComponent(portfolio.id)}`}
            className="block rounded-lg border border-slate-200 bg-white p-4 shadow-sm transition-colors hover:border-slate-400"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h2 className="font-semibold">{portfolio.name}</h2>
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium">
                {portfolio.status}
              </span>
            </div>
            <p className="mt-2 text-sm text-slate-600">
              Base currency: {portfolio.baseCurrency} · Updated{" "}
              {formatTimestamp(portfolio.updatedAt)}
            </p>
          </Link>
        </li>
      ))}
    </ul>
  );
}

export function formatTimestamp(value: string): string {
  const timestamp = new Date(value);
  if (Number.isNaN(timestamp.getTime())) return value;
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(timestamp);
}
