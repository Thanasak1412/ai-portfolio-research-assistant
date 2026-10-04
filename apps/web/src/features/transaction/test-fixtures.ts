import type { Asset } from "@/features/asset/api/asset-api";
import type {
  Transaction,
  TransactionCommand,
  TransactionCorrectionResult,
} from "@/features/transaction/api/transaction-api";

export const command = {
  kind: "DEPOSIT",
  amount: "10.000",
  currency: "USD",
  effectiveAt: "2020-01-02T12:30:00.123456Z",
} satisfies TransactionCommand;
export const transaction: Transaction = {
  id: "transaction-1",
  kind: "DEPOSIT",
  assetId: null,
  quantity: null,
  unitPrice: null,
  fee: null,
  amount: "10",
  currency: "USD",
  effectiveAt: command.effectiveAt,
  portfolioSequence: "12345678901234567890",
  note: null,
  externalReference: null,
  createdAt: "2020-01-02T12:30:01Z",
  correctionLinks: {
    reversesTransactionId: null,
    replacesTransactionId: null,
    reversalTransactionId: null,
    replacementTransactionId: null,
  },
};
export const asset: Asset = {
  id: "asset-1",
  symbol: "SYNTH",
  name: "Synthetic Equity",
  assetType: "EQUITY",
  exchange: "NYSE",
  currency: "USD",
};
export const correction: TransactionCorrectionResult = {
  original: {
    ...transaction,
    correctionLinks: {
      ...transaction.correctionLinks,
      reversalTransactionId: "reversal-1",
      replacementTransactionId: "replacement-1",
    },
  },
  reversal: {
    ...transaction,
    id: "reversal-1",
    kind: "REVERSAL",
    portfolioSequence: "12345678901234567891",
    correctionLinks: {
      ...transaction.correctionLinks,
      reversesTransactionId: transaction.id,
    },
  },
  replacement: {
    ...transaction,
    id: "replacement-1",
    amount: "20.123456789012",
    portfolioSequence: "12345678901234567892",
    correctionLinks: {
      ...transaction.correctionLinks,
      replacesTransactionId: transaction.id,
    },
  },
};
export const portfolio = {
  id: "portfolio-1",
  name: "Synthetic Portfolio",
  baseCurrency: "USD",
  status: "ACTIVE",
  archivedAt: null,
  createdAt: "2020-01-01T00:00:00Z",
  updatedAt: "2020-01-01T00:00:00Z",
} as const;
