"use client";

import { useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import type { Asset } from "@/features/asset/api/asset-api";
import type {
  Transaction,
  TransactionCommand,
  TransactionCorrectionResult,
} from "@/features/transaction/api/transaction-api";
import { TransactionCommandEditor } from "@/features/transaction/components/transaction-command-editor";
import { TransactionFacts } from "@/features/transaction/components/transaction-facts";
import { TransactionError } from "@/features/transaction/components/transaction-error";
import { CorrectionAttempt } from "@/features/transaction/model/command-attempt";
import { useCorrectTransaction } from "@/features/transaction/model/transaction-queries";
import { transactionKeys } from "@/features/transaction/model/transaction-query-keys";
import { ApiError } from "@/platform/api/api-error";

function focusHeading(node: HTMLHeadingElement | null) {
  node?.focus();
}

export function TransactionCorrectionForm({
  portfolioId,
  original,
  onClose,
}: Readonly<{
  portfolioId: string;
  original: Transaction;
  onClose: () => void;
}>) {
  const correct = useCorrectTransaction(portfolioId);
  const client = useQueryClient();
  const attempt = useRef(new CorrectionAttempt());
  const submitting = useRef(false);
  const [pending, setPending] = useState(false);
  const [review, setReview] = useState<TransactionCommand | null>(null);
  const [asset, setAsset] = useState<Asset | null>(null);
  const [accepted, setAccepted] = useState<TransactionCorrectionResult | null>(
    null,
  );
  const [error, setError] = useState<unknown>(null);
  const uncertain =
    error !== null &&
    (!(error instanceof ApiError) ||
      error.status === 0 ||
      error.status >= 500 ||
      error.code === "INTERNAL_ERROR");
  const terminal =
    error instanceof ApiError &&
    [
      "TRANSACTION_ALREADY_CORRECTED",
      "TRANSACTION_NOT_CORRECTABLE",
      "TRANSACTION_NOT_FOUND",
      "PORTFOLIO_NOT_FOUND",
      "PORTFOLIO_ARCHIVED",
    ].includes(error.code);

  async function confirm() {
    if (!review || submitting.current || terminal) return;
    submitting.current = true;
    setPending(true);
    setError(null);
    try {
      const result = await correct.mutateAsync({
        transactionId: original.id,
        command: { replacement: review },
        key: attempt.current.keyFor(original.id, review),
      });
      attempt.current.clear();
      setAccepted(result);
      setReview(null);
      correct.reset();
    } catch (failure) {
      setError(failure);
    } finally {
      submitting.current = false;
      setPending(false);
    }
  }
  function inspectHistory() {
    void client.invalidateQueries({
      queryKey: transactionKeys.histories(portfolioId),
    });
    void client.invalidateQueries({
      queryKey: transactionKeys.details(portfolioId),
    });
    document.getElementById("history-title")?.focus();
  }
  return (
    <section
      aria-labelledby="correction-title"
      className="space-y-4 rounded-lg border bg-white p-5"
    >
      <h2
        id="correction-title"
        tabIndex={-1}
        ref={focusHeading}
        className="text-xl font-semibold"
      >
        Correct Transaction
      </h2>
      <p>
        The original Transaction will not be edited or deleted. A correction
        records an internal reversal and a new replacement Transaction
        atomically.
      </p>
      {accepted ? (
        <div className="space-y-4">
          <h3 tabIndex={-1} ref={focusHeading} className="font-semibold">
            Correction accepted
          </h3>
          <p role="status">
            Original, reversal, and replacement confirmed by the server.
          </p>
          {(["original", "reversal", "replacement"] as const).map((role) => (
            <section
              key={role}
              aria-label={role}
              className="space-y-2 rounded border p-3"
            >
              <h4 className="font-semibold">
                {role === "original"
                  ? "Original — retained"
                  : role === "reversal"
                    ? "Reversal — internal record"
                    : "Replacement — new record"}
              </h4>
              <p className="break-all">{accepted[role].id}</p>
              <TransactionFacts transaction={accepted[role]} />
            </section>
          ))}
        </div>
      ) : (
        <>
          <section aria-labelledby="correction-original-title">
            <h3 id="correction-original-title" className="font-semibold">
              Original — remains in history
            </h3>
            <p className="break-all">{original.id}</p>
            <TransactionFacts transaction={original} />
          </section>
          <div hidden={review !== null}>
            <h3 className="font-semibold">Complete replacement</h3>
            <p>
              Enter all replacement facts below. No original fields are silently
              copied; select an eligible Asset again when required.
            </p>
          </div>
          <TransactionCommandEditor
            mode="replacement"
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
                Review correction
              </h3>
              <section aria-labelledby="correction-replacement-title">
                <h4 id="correction-replacement-title" className="font-semibold">
                  Replacement — new facts to be recorded
                </h4>
                {asset && (
                  <p>
                    {asset.symbol} — {asset.name} · {asset.assetType} ·{" "}
                    {asset.exchange}
                  </p>
                )}
                <TransactionFacts transaction={review} />
              </section>
              <p>
                The server creates the internal reversal and replacement in one
                atomic command. No partial result is accepted.
              </p>
              {error !== null && <TransactionError error={error} />}
              {uncertain && (
                <p>
                  Acceptance may be uncertain. Retry same correction with the
                  same memory-only key. Check ledger history before changing
                  facts or leaving; leaving discards this retry key.
                </p>
              )}
              {error !== null && !uncertain && !terminal && (
                <p>
                  The correction was rejected. Return to replacement and review
                  its facts before submitting again.
                </p>
              )}
              <div className="flex flex-wrap gap-3">
                {!terminal && (error === null || uncertain) && (
                  <Button
                    type="button"
                    disabled={pending}
                    onClick={() => void confirm()}
                  >
                    {pending
                      ? "Submitting correction…"
                      : uncertain
                        ? "Retry same correction"
                        : "Confirm correction"}
                  </Button>
                )}
                {!terminal && (
                  <Button
                    type="button"
                    variant="outline"
                    disabled={pending}
                    onClick={() => {
                      setReview(null);
                      setError(null);
                    }}
                  >
                    Return to replacement
                  </Button>
                )}
              </div>
            </div>
          )}
        </>
      )}
      <div className="flex flex-wrap gap-3">
        <Button
          type="button"
          variant="outline"
          disabled={pending}
          onClick={inspectHistory}
        >
          Refresh ledger history
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={pending}
          onClick={onClose}
        >
          Return to history
        </Button>
      </div>
    </section>
  );
}
