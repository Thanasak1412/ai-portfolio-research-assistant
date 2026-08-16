import type { PortfolioStatus } from "@/features/portfolio/api/portfolio-api";

export const portfolioKeys = {
  all: ["portfolios"] as const,
  lists: () => [...portfolioKeys.all, "list"] as const,
  list: (status: PortfolioStatus) =>
    [...portfolioKeys.lists(), status] as const,
  details: () => [...portfolioKeys.all, "detail"] as const,
  detail: (id: string) => [...portfolioKeys.details(), id] as const,
};
