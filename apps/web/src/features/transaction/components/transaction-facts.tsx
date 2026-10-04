import type {
  Transaction,
  TransactionCommand,
} from "@/features/transaction/api/transaction-api";

export function TransactionFacts({
  transaction,
}: Readonly<{ transaction: Transaction | TransactionCommand }>) {
  const facts: [string, string | null | undefined][] = [
    ["Kind", transaction.kind],
    ["Asset ID", "assetId" in transaction ? transaction.assetId : null],
    ["Quantity", "quantity" in transaction ? transaction.quantity : null],
    ["Unit price", "unitPrice" in transaction ? transaction.unitPrice : null],
    [
      "Fee",
      "quantity" in transaction && transaction.quantity !== null
        ? (transaction.fee ?? "0 (omitted fee)")
        : null,
    ],
    ["Amount", "amount" in transaction ? transaction.amount : null],
    ["Currency", transaction.currency],
    ["Effective time (UTC)", transaction.effectiveAt],
    [
      "Portfolio sequence",
      "portfolioSequence" in transaction ? transaction.portfolioSequence : null,
    ],
    ["Note", transaction.note],
    ["External reference", transaction.externalReference],
  ];
  return (
    <dl className="grid gap-3 text-sm sm:grid-cols-2">
      {facts
        .filter(([, value]) => value !== null && value !== undefined)
        .map(([label, value]) => (
          <div key={label} className="min-w-0">
            <dt className="text-slate-600">{label}</dt>
            <dd className="whitespace-pre-wrap break-words font-medium">
              {value === "" ? "(empty string)" : value}
            </dd>
          </div>
        ))}
    </dl>
  );
}
