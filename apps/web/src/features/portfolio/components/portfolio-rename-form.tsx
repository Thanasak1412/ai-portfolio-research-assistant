"use client";

import { useQueryClient } from "@tanstack/react-query";
import type { FormEvent } from "react";
import { useState, useSyncExternalStore } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { Portfolio } from "@/features/portfolio/api/portfolio-api";
import { portfolioErrorMessage } from "@/features/portfolio/components/portfolio-error";
import { useUpdatePortfolio } from "@/features/portfolio/model/portfolio-queries";
import { portfolioKeys } from "@/features/portfolio/model/portfolio-query-keys";
import { portfolioNameFormSchema } from "@/features/portfolio/model/portfolio-validation";
import { ApiError } from "@/platform/api/api-error";

function subscribeToNothing() {
  return () => {};
}

function getClientSnapshot() {
  return true;
}

function getServerSnapshot() {
  return false;
}

export function PortfolioRenameForm({
  portfolio,
  name,
  onNameChange,
}: Readonly<{
  portfolio: Portfolio;
  name: string;
  onNameChange: (name: string) => void;
}>) {
  const updatePortfolio = useUpdatePortfolio();
  const queryClient = useQueryClient();
  const [submissionError, setSubmissionError] = useState<string | null>(null);
  const [validationError, setValidationError] = useState<string | null>(null);
  const isHydrated = useSyncExternalStore(
    subscribeToNothing,
    getClientSnapshot,
    getServerSnapshot,
  );

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmissionError(null);
    const values = portfolioNameFormSchema.safeParse({
      name: new FormData(event.currentTarget).get("name"),
    });
    if (!values.success) {
      setValidationError(values.error.issues[0]?.message ?? "Invalid name.");
      return;
    }
    setValidationError(null);
    try {
      await updatePortfolio.mutateAsync({
        portfolioId: portfolio.id,
        input: { name: values.data.name },
      });
    } catch (error) {
      if (error instanceof ApiError && error.code === "PORTFOLIO_ARCHIVED") {
        await queryClient.invalidateQueries({
          queryKey: portfolioKeys.detail(portfolio.id),
        });
      }
      setSubmissionError(portfolioErrorMessage(error));
    }
  };

  return (
    <section
      aria-labelledby="rename-portfolio-title"
      className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 id="rename-portfolio-title" className="text-lg font-semibold">
        Rename Portfolio
      </h2>
      <form
        className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-start"
        onSubmit={submit}
        noValidate
      >
        <div className="min-w-0 flex-1">
          <label
            htmlFor="rename-portfolio-name"
            className="text-sm font-medium"
          >
            Portfolio name
          </label>
          <Input
            id="rename-portfolio-name"
            disabled={!isHydrated}
            aria-invalid={!!validationError}
            aria-describedby={
              validationError ? "rename-portfolio-name-error" : undefined
            }
            name="name"
            value={name}
            onChange={(event) => {
              onNameChange(event.currentTarget.value);
              setValidationError(null);
            }}
          />
          {validationError && (
            <p
              id="rename-portfolio-name-error"
              role="alert"
              className="mt-1 text-sm text-red-700"
            >
              {validationError}
            </p>
          )}
        </div>
        <Button
          type="submit"
          disabled={!isHydrated || updatePortfolio.isPending}
          className="sm:mt-6"
        >
          {updatePortfolio.isPending ? "Saving…" : "Save name"}
        </Button>
      </form>
      {submissionError && (
        <p role="alert" className="mt-3 text-sm text-red-700">
          {submissionError}
        </p>
      )}
    </section>
  );
}
