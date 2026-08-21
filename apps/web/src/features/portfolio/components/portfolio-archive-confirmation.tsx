"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import type { Portfolio } from "@/features/portfolio/api/portfolio-api";
import { portfolioErrorMessage } from "@/features/portfolio/components/portfolio-error";
import { useArchivePortfolio } from "@/features/portfolio/model/portfolio-queries";

export function PortfolioArchiveConfirmation({
  portfolio,
}: Readonly<{ portfolio: Portfolio }>) {
  const [confirming, setConfirming] = useState(false);
  const [submissionError, setSubmissionError] = useState<string | null>(null);
  const archivePortfolio = useArchivePortfolio();
  if (!confirming)
    return (
      <Button
        type="button"
        variant="outline"
        onClick={() => setConfirming(true)}
      >
        Archive Portfolio
      </Button>
    );

  const archive = async () => {
    setSubmissionError(null);
    try {
      await archivePortfolio.mutateAsync(portfolio.id);
      setConfirming(false);
    } catch (error) {
      setSubmissionError(portfolioErrorMessage(error));
    }
  };
  return (
    <section
      aria-labelledby="archive-portfolio-title"
      className="rounded-lg border border-amber-300 bg-amber-50 p-5"
    >
      <h2 id="archive-portfolio-title" className="font-semibold">
        Archive this Portfolio?
      </h2>
      <p className="mt-2 text-sm text-slate-700">
        The Portfolio record is retained, becomes read-only in M2, and its name
        may later be reused for another active Portfolio.
      </p>
      <div className="mt-4 flex gap-3">
        <Button
          type="button"
          variant="outline"
          disabled={archivePortfolio.isPending}
          onClick={() => setConfirming(false)}
        >
          Cancel
        </Button>
        <Button
          type="button"
          disabled={archivePortfolio.isPending}
          onClick={() => void archive()}
        >
          {archivePortfolio.isPending ? "Archiving…" : "Archive Portfolio"}
        </Button>
      </div>
      {submissionError && (
        <p role="alert" className="mt-3 text-sm text-red-700">
          {submissionError}
        </p>
      )}
    </section>
  );
}
