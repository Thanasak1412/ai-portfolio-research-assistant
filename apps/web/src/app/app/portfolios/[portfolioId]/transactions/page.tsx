import { TransactionScreen } from "@/features/transaction/components/transaction-screen";

export default async function TransactionsPage({
  params,
}: Readonly<{ params: Promise<{ portfolioId: string }> }>) {
  const { portfolioId } = await params;
  return <TransactionScreen portfolioId={portfolioId} />;
}
