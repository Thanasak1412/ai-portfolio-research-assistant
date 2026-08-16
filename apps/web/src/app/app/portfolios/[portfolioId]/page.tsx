import { PortfolioDetailScreen } from "@/features/portfolio/components/portfolio-detail-screen";

export default async function PortfolioDetailPage({
  params,
}: Readonly<{ params: Promise<{ portfolioId: string }> }>) {
  const { portfolioId } = await params;
  return <PortfolioDetailScreen portfolioId={portfolioId} />;
}
