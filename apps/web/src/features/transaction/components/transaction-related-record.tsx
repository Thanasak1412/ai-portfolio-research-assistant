"use client";

import { Button } from "@/components/ui/button";
import type { Transaction } from "@/features/transaction/api/transaction-api";
import { useTransaction } from "@/features/transaction/model/transaction-queries";
import { TransactionError } from "@/features/transaction/components/transaction-error";
import { TransactionRecord } from "@/features/transaction/components/transaction-record";

function focusHeading(node: HTMLHeadingElement | null) {
  node?.focus();
}

export function TransactionRelatedRecord({
  portfolioId,
  transactionId,
  loadedIds,
  onNavigate,
  onCorrect,
  correctionPending,
  onClose,
}: Readonly<{
  portfolioId: string;
  transactionId: string;
  loadedIds: Set<string>;
  onNavigate: (id: string) => void;
  onCorrect?: (transaction: Transaction) => void;
  correctionPending: boolean;
  onClose: () => void;
}>) {
  const detail = useTransaction(portfolioId, transactionId);
  return (
    <section
      aria-labelledby="related-title"
      className="space-y-3 rounded border bg-white p-4"
    >
      <h2
        id="related-title"
        tabIndex={-1}
        ref={focusHeading}
        className="font-semibold"
      >
        Related Transaction
      </h2>
      {detail.isLoading && <p role="status">Loading related Transaction…</p>}
      {detail.isError ? (
        <>
          <TransactionError error={detail.error} />
          <Button onClick={() => void detail.refetch()}>
            Retry related Transaction
          </Button>
        </>
      ) : (
        detail.data && (
          <TransactionRecord
            transaction={detail.data}
            loadedIds={loadedIds}
            onNavigate={onNavigate}
            onCorrect={onCorrect}
            correctionPending={correctionPending}
          />
        )
      )}
      <Button variant="outline" onClick={onClose}>
        Close related Transaction
      </Button>
    </section>
  );
}
