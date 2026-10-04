import { Button } from "@/components/ui/button";
import type { Transaction } from "@/features/transaction/api/transaction-api";
import { TransactionFacts } from "@/features/transaction/components/transaction-facts";

export const transactionAnchor = (id: string) =>
  `transaction-${encodeURIComponent(id)}`;
export function canCorrect(transaction: Transaction): boolean {
  return (
    transaction.kind !== "REVERSAL" &&
    transaction.correctionLinks.reversalTransactionId === null
  );
}

export function TransactionRecord({
  transaction,
  loadedIds,
  onNavigate,
  onCorrect,
  correctionPending = false,
}: Readonly<{
  transaction: Transaction;
  loadedIds: Set<string>;
  onNavigate: (id: string) => void;
  onCorrect?: (transaction: Transaction) => void;
  correctionPending?: boolean;
}>) {
  const links = [
    ["Reverses", transaction.correctionLinks.reversesTransactionId],
    ["Replaces", transaction.correctionLinks.replacesTransactionId],
    ["Reversal", transaction.correctionLinks.reversalTransactionId],
    ["Replacement", transaction.correctionLinks.replacementTransactionId],
  ] as const;
  return (
    <>
      <h3 className="break-all font-semibold">
        {transaction.kind} · {transaction.id}
      </h3>
      <TransactionFacts transaction={transaction} />
      <ul className="space-y-2" aria-label="Correction relationships">
        {links
          .filter(([, id]) => id !== null)
          .map(([label, id]) => (
            <li key={label} className="break-all">
              {loadedIds.has(id!) ? (
                <a
                  className="underline"
                  href={`#${transactionAnchor(id!)}`}
                  onClick={() =>
                    document.getElementById(transactionAnchor(id!))?.focus()
                  }
                >
                  {label}: {id}
                </a>
              ) : (
                <Button
                  type="button"
                  variant="outline"
                  className="h-auto whitespace-normal text-left"
                  onClick={() => onNavigate(id!)}
                >
                  {label}: {id}
                </Button>
              )}
            </li>
          ))}
      </ul>
      {onCorrect && canCorrect(transaction) && (
        <Button
          type="button"
          disabled={correctionPending}
          onClick={() => onCorrect(transaction)}
        >
          Correct
        </Button>
      )}
    </>
  );
}
