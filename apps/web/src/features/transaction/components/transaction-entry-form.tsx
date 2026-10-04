"use client";

import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import type { Asset } from "@/features/asset/api/asset-api";
import type {
  Transaction,
  TransactionCommand,
} from "@/features/transaction/api/transaction-api";
import { CommandAttempt } from "@/features/transaction/model/command-attempt";
import { useCreateTransaction } from "@/features/transaction/model/transaction-queries";
import { TransactionCommandEditor } from "@/features/transaction/components/transaction-command-editor";
import { TransactionError } from "@/features/transaction/components/transaction-error";
import { TransactionFacts } from "@/features/transaction/components/transaction-facts";

function focusHeading(node: HTMLHeadingElement | null) {
  node?.focus();
}

export function TransactionEntryForm({
  portfolioId,
}: Readonly<{ portfolioId: string }>) {
  const create = useCreateTransaction(portfolioId);
  const [review, setReview] = useState<TransactionCommand | null>(null);
  const [asset, setAsset] = useState<Asset | null>(null);
  const [accepted, setAccepted] = useState<Transaction | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [pending, setPending] = useState(false);
  const [focusNextEntry, setFocusNextEntry] = useState(false);
  const attempt = useRef(new CommandAttempt());
  const submitting = useRef(false);
  async function confirm() {
    if (!review || submitting.current) return;
    submitting.current = true;
    setPending(true);
    setError(null);
    try {
      const result = await create.mutateAsync({
        command: review,
        key: attempt.current.keyFor(review),
      });
      attempt.current.clear();
      setAccepted(result);
      setReview(null);
      setAsset(null);
      create.reset();
    } catch (failure) {
      setError(failure);
    } finally {
      submitting.current = false;
      setPending(false);
    }
  }
  return (
    <section
      aria-labelledby="entry-title"
      className="space-y-4 rounded-lg border bg-white p-5"
    >
      <h2 id="entry-title" className="text-xl font-semibold">
        Record a Transaction
      </h2>
      {accepted ? (
        <div className="space-y-4">
          <h3
            tabIndex={-1}
            ref={focusHeading}
            role="status"
            className="font-semibold"
          >
            Transaction accepted
          </h3>
          <p>Immutable record: {accepted.id}</p>
          <TransactionFacts transaction={accepted} />
          <Button
            type="button"
            onClick={() => {
              setFocusNextEntry(true);
              setAccepted(null);
            }}
          >
            Record another entry
          </Button>
        </div>
      ) : (
        <>
          <TransactionCommandEditor
            focusOnMount={focusNextEntry}
            hidden={review !== null}
            onReview={(command, selected) => {
              setReview(command);
              setAsset(selected);
              setError(null);
            }}
          />
          {review && (
            <div className="space-y-4">
              <h3 tabIndex={-1} ref={focusHeading} className="font-semibold">
                Review entry before submitting
              </h3>
              {asset && (
                <p>
                  {asset.symbol} — {asset.name} · {asset.assetType} ·{" "}
                  {asset.exchange}
                </p>
              )}
              <TransactionFacts transaction={review} />
              <p className="text-sm">
                Confirm these exact facts. The server decides eligibility and
                ledger validity.
              </p>
              {error !== null && (
                <>
                  <TransactionError error={error} />
                  <p>
                    Acceptance may be uncertain. Retry unchanged to reuse this
                    attempt. Before changing facts or leaving, check history;
                    this attempt is held only in memory.
                  </p>
                </>
              )}
              <div className="flex gap-3">
                <Button
                  type="button"
                  disabled={pending}
                  onClick={() => void confirm()}
                >
                  {pending
                    ? "Submitting…"
                    : error !== null
                      ? "Retry same entry"
                      : "Confirm submission"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={pending}
                  onClick={() => {
                    setReview(null);
                    setError(null);
                  }}
                >
                  Return to entry
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </section>
  );
}
