"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import type { PortfolioStatus } from "@/features/portfolio/api/portfolio-api";
import { PortfolioCreateForm } from "@/features/portfolio/components/portfolio-create-form";
import { PortfolioError } from "@/features/portfolio/components/portfolio-error";
import { PortfolioList } from "@/features/portfolio/components/portfolio-list";
import { usePortfolios } from "@/features/portfolio/model/portfolio-queries";

export function PortfolioListScreen() {
  const [status, setStatus] = useState<PortfolioStatus>("ACTIVE");
  const portfolios = usePortfolios(status);
  return (
    <main className="mx-auto max-w-3xl space-y-6 px-4 py-8 sm:px-6">
      <header>
        <h1 className="text-2xl font-semibold">Portfolios</h1>
        <p className="mt-1 text-sm text-slate-600">
          Create and manage your Portfolio records.
        </p>
      </header>
      <PortfolioCreateForm />
      <div className="flex gap-2" aria-label="Portfolio status">
        <StatusButton status="ACTIVE" selected={status} onSelect={setStatus} />
        <StatusButton
          status="ARCHIVED"
          selected={status}
          onSelect={setStatus}
        />
      </div>
      {portfolios.isLoading ? <p role="status">Loading portfolios…</p> : null}
      {portfolios.isError ? (
        <div className="space-y-3">
          <PortfolioError error={portfolios.error} />
          <Button
            type="button"
            variant="outline"
            onClick={() => void portfolios.refetch()}
          >
            Retry
          </Button>
        </div>
      ) : null}
      {portfolios.isSuccess && portfolios.data.items.length === 0 ? (
        <EmptyState status={status} />
      ) : null}
      {portfolios.isSuccess && portfolios.data.items.length > 0 ? (
        <PortfolioList portfolios={portfolios.data.items} />
      ) : null}
    </main>
  );
}

function StatusButton({
  status,
  selected,
  onSelect,
}: Readonly<{
  status: PortfolioStatus;
  selected: PortfolioStatus;
  onSelect: (next: PortfolioStatus) => void;
}>) {
  return (
    <Button
      type="button"
      variant={selected === status ? "default" : "outline"}
      aria-pressed={selected === status}
      onClick={() => onSelect(status)}
    >
      {status === "ACTIVE" ? "Active" : "Archived"}
    </Button>
  );
}

function EmptyState({ status }: Readonly<{ status: PortfolioStatus }>) {
  return (
    <section className="rounded-lg border border-dashed border-slate-300 p-6 text-sm text-slate-600">
      <p>
        {status === "ACTIVE"
          ? "No active portfolios yet."
          : "No archived portfolios."}
      </p>
    </section>
  );
}
