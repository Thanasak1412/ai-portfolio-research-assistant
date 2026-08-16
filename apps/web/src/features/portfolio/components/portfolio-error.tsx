import { ApiError } from "@/platform/api/api-error";

export function portfolioErrorMessage(error: unknown): string {
  if (!(error instanceof ApiError)) {
    return "Portfolio data is temporarily unavailable. Please try again.";
  }
  switch (error.code) {
    case "PORTFOLIO_NAME_CONFLICT":
      return "An active portfolio with this name already exists.";
    case "PORTFOLIO_NOT_FOUND":
      return "Portfolio not found.";
    case "PORTFOLIO_ARCHIVED":
      return "This portfolio is archived and can no longer be edited.";
    default:
      return "Portfolio data is temporarily unavailable. Please try again.";
  }
}

export function PortfolioError({ error }: Readonly<{ error: unknown }>) {
  const correlationId =
    error instanceof ApiError ? error.correlationId : undefined;
  return (
    <div
      role="alert"
      className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-800"
    >
      <p>{portfolioErrorMessage(error)}</p>
      {correlationId && (
        <p className="mt-1 text-xs">Reference: {correlationId}</p>
      )}
    </div>
  );
}
