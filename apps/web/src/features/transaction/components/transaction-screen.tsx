"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { usePortfolio } from "@/features/portfolio/model/portfolio-queries";
import { TransactionEntryForm } from "@/features/transaction/components/transaction-entry-form";
import { TransactionHistory } from "@/features/transaction/components/transaction-history";
import { TransactionError } from "@/features/transaction/components/transaction-error";

export function TransactionScreen({
  portfolioId,
}: Readonly<{ portfolioId: string }>) {
  const portfolio = usePortfolio(portfolioId);
  return (
    <main className="mx-auto max-w-4xl space-y-6 px-4 py-8 sm:px-6">
      <Button asChild variant="outline">
        <Link href={`/app/portfolios/${encodeURIComponent(portfolioId)}`}>
          Back to Portfolio
        </Link>
      </Button>
      <h1 className="text-2xl font-semibold">Transactions</h1>
      {portfolio.isLoading && <p role="status">Loading portfolio…</p>}
      {portfolio.isError ? (
        <>
          <TransactionError error={portfolio.error} />
          <Button variant="outline" onClick={() => void portfolio.refetch()}>
            Retry Portfolio
          </Button>
        </>
      ) : (
        portfolio.data && (
          <>
            <p className="font-medium">
              {portfolio.data.name} · {portfolio.data.status}
            </p>
            {portfolio.data.status === "ACTIVE" ? (
              <TransactionEntryForm
                key={`entry-${portfolioId}`}
                portfolioId={portfolioId}
              />
            ) : (
              <p>This Portfolio is archived. History is read-only.</p>
            )}
            <TransactionHistory
              key={`history-${portfolioId}`}
              portfolioId={portfolioId}
            />
          </>
        )
      )}
    </main>
  );
}
