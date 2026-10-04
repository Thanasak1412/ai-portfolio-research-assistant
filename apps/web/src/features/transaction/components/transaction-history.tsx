"use client";

import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type {
  Transaction,
  TransactionFilters,
} from "@/features/transaction/api/transaction-api";
import { useTransactionHistory } from "@/features/transaction/model/transaction-queries";
import {
  historyFiltersSchema,
  visibleKinds,
} from "@/features/transaction/model/transaction-validation";
import { TransactionError } from "@/features/transaction/components/transaction-error";
import {
  TransactionRecord,
  transactionAnchor,
} from "@/features/transaction/components/transaction-record";
import { TransactionRelatedRecord } from "@/features/transaction/components/transaction-related-record";

export function TransactionHistory({
  portfolioId,
  onCorrect,
  correctionPending = false,
}: Readonly<{
  portfolioId: string;
  onCorrect?: (transaction: Transaction) => void;
  correctionPending?: boolean;
}>) {
  const [relatedId, setRelatedId] = useState<string | null>(null);
  const [filters, setFilters] = useState<TransactionFilters>({
    includeReversals: true,
  });
  const [filterError, setFilterError] = useState<string | null>(null);
  const history = useTransactionHistory(portfolioId, filters);
  const rows = history.data?.pages.flatMap((page) => page.items) ?? [];
  const loadedIds = new Set(rows.map((row) => row.id));

  function apply(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const parsed = historyFiltersSchema.safeParse({
      kind: form.get("kind") || undefined,
      effectiveAtFrom: form.get("from") || undefined,
      effectiveAtTo: form.get("to") || undefined,
      includeReversals: form.has("reversals"),
    });
    if (!parsed.success) {
      setFilterError(parsed.error.issues[0].message);
      return;
    }
    setFilterError(null);
    setFilters(parsed.data);
  }

  return (
    <section aria-labelledby="history-title" className="space-y-4">
      <header>
        <h2 id="history-title" tabIndex={-1} className="text-xl font-semibold">
          Ledger history
        </h2>
        <p className="text-sm text-slate-600">
          Immutable server records, newest effective time first. No client-side
          financial calculations.
        </p>
      </header>
      <form
        onSubmit={apply}
        onReset={() => {
          setFilters({ includeReversals: true });
          setFilterError(null);
        }}
        className="space-y-3 rounded-lg border p-4"
        noValidate
      >
        <fieldset className="grid gap-3 sm:grid-cols-3">
          <legend className="font-medium">History filters</legend>
          <div>
            <label htmlFor="history-kind">Kind</label>
            <select
              name="kind"
              id="history-kind"
              defaultValue=""
              className="block rounded border p-2"
            >
              <option value="">All kinds</option>
              {visibleKinds.map((kind) => (
                <option key={kind}>{kind}</option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="history-from">
              Effective from (UTC, inclusive)
            </label>
            <Input
              id="history-from"
              name="from"
              placeholder="2026-01-01T00:00:00Z"
              aria-describedby="history-time-help"
            />
          </div>
          <div>
            <label htmlFor="history-to">Effective to (UTC, inclusive)</label>
            <Input
              id="history-to"
              name="to"
              placeholder="2026-12-31T23:59:59Z"
              aria-describedby="history-time-help"
            />
          </div>
        </fieldset>
        <p id="history-time-help" className="text-sm">
          Use UTC-Z timestamps. Past and future filter bounds are allowed.
        </p>
        <label className="block">
          <input type="checkbox" name="reversals" defaultChecked /> Include
          internal reversals
        </label>
        {filterError && <p role="alert">{filterError}</p>}
        <div className="flex gap-2">
          <Button type="submit">Apply filters</Button>
          <Button type="reset" variant="outline">
            Clear filters
          </Button>
        </div>
      </form>
      {relatedId && (
        <TransactionRelatedRecord
          key={relatedId}
          portfolioId={portfolioId}
          transactionId={relatedId}
          loadedIds={loadedIds}
          onNavigate={setRelatedId}
          onCorrect={onCorrect}
          correctionPending={correctionPending}
          onClose={() => {
            setRelatedId(null);
            document.getElementById("history-title")?.focus();
          }}
        />
      )}
      {history.isLoading && <p role="status">Loading transactions…</p>}
      {history.isError && (
        <div className="space-y-2">
          <TransactionError error={history.error} />
          <Button
            variant="outline"
            onClick={() =>
              void (history.isFetchNextPageError
                ? history.fetchNextPage()
                : history.refetch())
            }
          >
            Retry history
          </Button>
        </div>
      )}
      {history.isSuccess && rows.length === 0 && (
        <p>No transactions match these filters.</p>
      )}
      <ol className="space-y-4">
        {rows.map((transaction) => (
          <li
            key={transaction.id}
            id={transactionAnchor(transaction.id)}
            tabIndex={-1}
            className="space-y-3 rounded-lg border bg-white p-4"
          >
            <TransactionRecord
              transaction={transaction}
              loadedIds={loadedIds}
              onNavigate={setRelatedId}
              onCorrect={onCorrect}
              correctionPending={correctionPending}
            />
          </li>
        ))}
      </ol>
      {history.hasNextPage && (
        <Button
          type="button"
          variant="outline"
          disabled={history.isFetching}
          onClick={() => void history.fetchNextPage()}
        >
          {history.isFetchingNextPage
            ? "Loading more…"
            : "Load more transactions"}
        </Button>
      )}
    </section>
  );
}
