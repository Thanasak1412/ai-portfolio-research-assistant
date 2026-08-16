"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { Portfolio } from "@/features/portfolio/api/portfolio-api";
import { portfolioErrorMessage } from "@/features/portfolio/components/portfolio-error";
import { useUpdatePortfolio } from "@/features/portfolio/model/portfolio-queries";
import { portfolioKeys } from "@/features/portfolio/model/portfolio-query-keys";
import { ApiError } from "@/platform/api/api-error";
import {
  portfolioNameFormSchema,
  type PortfolioNameFormValues,
} from "@/features/portfolio/model/portfolio-validation";

export function PortfolioRenameForm({
  portfolio,
}: Readonly<{ portfolio: Portfolio }>) {
  const updatePortfolio = useUpdatePortfolio();
  const queryClient = useQueryClient();
  const [submissionError, setSubmissionError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<PortfolioNameFormValues>({
    resolver: zodResolver(portfolioNameFormSchema),
    defaultValues: { name: portfolio.name },
  });
  useEffect(() => reset({ name: portfolio.name }), [portfolio.name, reset]);

  const submit = async (values: PortfolioNameFormValues) => {
    setSubmissionError(null);
    try {
      await updatePortfolio.mutateAsync({
        portfolioId: portfolio.id,
        input: { name: values.name },
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
        onSubmit={handleSubmit(submit)}
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
            aria-invalid={!!errors.name}
            aria-describedby={
              errors.name ? "rename-portfolio-name-error" : undefined
            }
            {...register("name")}
          />
          {errors.name && (
            <p
              id="rename-portfolio-name-error"
              role="alert"
              className="mt-1 text-sm text-red-700"
            >
              {errors.name.message}
            </p>
          )}
        </div>
        <Button
          type="submit"
          disabled={updatePortfolio.isPending}
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
