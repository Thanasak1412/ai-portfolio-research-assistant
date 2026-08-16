"use client";

import Link from "next/link";

import { Button } from "@/components/ui/button";
import { ApiError } from "@/platform/api/api-error";
import { PortfolioArchiveConfirmation } from "@/features/portfolio/components/portfolio-archive-confirmation";
import { PortfolioError } from "@/features/portfolio/components/portfolio-error";
import { formatTimestamp } from "@/features/portfolio/components/portfolio-list";
import { PortfolioRenameForm } from "@/features/portfolio/components/portfolio-rename-form";
import { usePortfolio } from "@/features/portfolio/model/portfolio-queries";

export function PortfolioDetailScreen({
  portfolioId,
}: Readonly<{ portfolioId: string }>) {
  const portfolio = usePortfolio(portfolioId);
  if (portfolio.isLoading)
    return (
      <main className="mx-auto max-w-3xl px-4 py-8 sm:px-6">
        <p role="status">Loading portfolio…</p>
      </main>
    );
  if (
    portfolio.isError &&
    portfolio.error instanceof ApiError &&
    portfolio.error.code === "PORTFOLIO_NOT_FOUND"
  )
    return (
      <main className="mx-auto max-w-3xl space-y-4 px-4 py-8 sm:px-6">
        <h1 className="text-2xl font-semibold">Portfolio not found.</h1>
        <Button asChild variant="outline">
          <Link href="/app/portfolios">Back to Portfolios</Link>
        </Button>
      </main>
    );
  if (portfolio.isError)
    return (
      <main className="mx-auto max-w-3xl space-y-3 px-4 py-8 sm:px-6">
        <PortfolioError error={portfolio.error} />
        <Button
          type="button"
          variant="outline"
          onClick={() => void portfolio.refetch()}
        >
          Retry
        </Button>
      </main>
    );
  if (!portfolio.data) return null;
  const data = portfolio.data;
  const archived = data.status === "ARCHIVED";
  return (
    <main className="mx-auto max-w-3xl space-y-6 px-4 py-8 sm:px-6">
      <Button asChild variant="outline" size="sm">
        <Link href="/app/portfolios">Back to Portfolios</Link>
      </Button>
      <header>
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="text-2xl font-semibold">{data.name}</h1>
          <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium">
            {data.status}
          </span>
        </div>
        <p className="mt-2 text-sm text-slate-600">
          Base currency: {data.baseCurrency}
        </p>
      </header>
      <dl className="grid gap-3 rounded-lg border border-slate-200 bg-white p-5 text-sm shadow-sm sm:grid-cols-2">
        <Metadata label="Created" value={formatTimestamp(data.createdAt)} />
        <Metadata label="Updated" value={formatTimestamp(data.updatedAt)} />
        {data.archivedAt && (
          <Metadata label="Archived" value={formatTimestamp(data.archivedAt)} />
        )}
      </dl>
      {archived ? (
        <section className="rounded-lg border border-slate-200 bg-slate-50 p-5 text-sm text-slate-700">
          <h2 className="font-semibold">Archived Portfolio</h2>
          <p className="mt-1">This Portfolio is read-only.</p>
        </section>
      ) : (
        <>
          <PortfolioRenameForm portfolio={data} />
          <PortfolioArchiveConfirmation portfolio={data} />
        </>
      )}
    </main>
  );
}

function Metadata({
  label,
  value,
}: Readonly<{ label: string; value: string }>) {
  return (
    <div>
      <dt className="text-slate-500">{label}</dt>
      <dd className="mt-1 font-medium">{value}</dd>
    </div>
  );
}
