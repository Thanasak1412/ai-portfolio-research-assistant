"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { portfolioErrorMessage } from "@/features/portfolio/components/portfolio-error";
import { useCreatePortfolio } from "@/features/portfolio/model/portfolio-queries";
import {
  portfolioNameFormSchema,
  type PortfolioNameFormValues,
} from "@/features/portfolio/model/portfolio-validation";

export function PortfolioCreateForm() {
  const router = useRouter();
  const createPortfolio = useCreatePortfolio();
  const [submissionError, setSubmissionError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<PortfolioNameFormValues>({
    resolver: zodResolver(portfolioNameFormSchema),
    defaultValues: { name: "" },
  });

  const submit = async (values: PortfolioNameFormValues) => {
    setSubmissionError(null);
    try {
      const portfolio = await createPortfolio.mutateAsync({
        name: values.name,
        baseCurrency: "USD",
      });
      reset();
      router.push(`/app/portfolios/${encodeURIComponent(portfolio.id)}`);
    } catch (error) {
      setSubmissionError(portfolioErrorMessage(error));
    }
  };

  return (
    <section
      aria-labelledby="create-portfolio-title"
      className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 id="create-portfolio-title" className="text-lg font-semibold">
        Create Portfolio
      </h2>
      <p className="mt-1 text-sm text-slate-600">
        All Portfolios use USD as their fixed base currency.
      </p>
      <form
        className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-start"
        onSubmit={handleSubmit(submit)}
        noValidate
      >
        <div className="min-w-0 flex-1">
          <label htmlFor="portfolio-name" className="text-sm font-medium">
            Portfolio name
          </label>
          <Input
            id="portfolio-name"
            aria-invalid={!!errors.name}
            aria-describedby={errors.name ? "portfolio-name-error" : undefined}
            {...register("name")}
          />
          {errors.name && (
            <p
              id="portfolio-name-error"
              role="alert"
              className="mt-1 text-sm text-red-700"
            >
              {errors.name.message}
            </p>
          )}
        </div>
        <Button
          type="submit"
          disabled={createPortfolio.isPending}
          className="sm:mt-6"
        >
          {createPortfolio.isPending ? "Creating…" : "Create Portfolio"}
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
