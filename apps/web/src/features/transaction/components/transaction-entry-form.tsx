"use client";

import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { Asset } from "@/features/asset/api/asset-api";
import type {
  Transaction,
  TransactionCommand,
} from "@/features/transaction/api/transaction-api";
import { CommandAttempt } from "@/features/transaction/model/command-attempt";
import { useCreateTransaction } from "@/features/transaction/model/transaction-queries";
import {
  commandSchema,
  createKinds,
  isEligibleAsset,
} from "@/features/transaction/model/transaction-validation";
import { TransactionAssetPicker } from "@/features/transaction/components/transaction-asset-picker";
import { TransactionError } from "@/features/transaction/components/transaction-error";
import { TransactionFacts } from "@/features/transaction/components/transaction-facts";

type Fields = {
  effectiveAt: string;
  quantity?: string;
  unitPrice?: string;
  fee?: string;
  amount?: string;
  note?: string;
  externalReference?: string;
};
type Kind = TransactionCommand["kind"];

function focusHeading(node: HTMLHeadingElement | null) {
  node?.focus();
}

export function TransactionEntryForm({
  portfolioId,
}: Readonly<{ portfolioId: string }>) {
  const create = useCreateTransaction(portfolioId);
  const [kind, setKind] = useState<Kind>("BUY");
  const [asset, setAsset] = useState<Asset | null>(null);
  const [includeNote, setIncludeNote] = useState(false);
  const [includeReference, setIncludeReference] = useState(false);
  const [review, setReview] = useState<TransactionCommand | null>(null);
  const [accepted, setAccepted] = useState<Transaction | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [assetError, setAssetError] = useState<string | null>(null);
  const [bodyError, setBodyError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const attempt = useRef(new CommandAttempt());
  const submitting = useRef(false);
  const {
    register,
    handleSubmit,
    setError: setFieldError,
    clearErrors,
    unregister,
    reset,
    setFocus,
    formState: { errors },
  } = useForm<Fields>({
    shouldUnregister: true,
    defaultValues: { effectiveAt: "" },
  });
  const trade = kind === "BUY" || kind === "SELL";
  const needsAsset = trade || kind === "DIVIDEND";
  const kindInput = useRef<HTMLSelectElement>(null);
  const previousPhase = useRef("edit");
  const phase = accepted ? "accepted" : review ? "review" : "edit";
  useEffect(() => {
    if (phase === "edit" && previousPhase.current !== "edit")
      kindInput.current?.focus();
    previousPhase.current = phase;
  }, [phase]);

  function prepare(values: Fields) {
    clearErrors();
    setAssetError(null);
    setBodyError(null);
    setError(null);
    if (needsAsset && (!asset || !isEligibleAsset(asset))) {
      setAssetError("Select an eligible canonical Asset.");
      return;
    }
    const candidate = {
      kind,
      currency: "USD",
      effectiveAt: values.effectiveAt,
      ...(includeNote ? { note: values.note ?? "" } : {}),
      ...(includeReference
        ? { externalReference: values.externalReference ?? "" }
        : {}),
      ...(needsAsset ? { assetId: asset?.id } : {}),
      ...(trade
        ? {
            quantity: values.quantity,
            unitPrice: values.unitPrice,
            ...(values.fee ? { fee: values.fee } : {}),
          }
        : { amount: values.amount }),
    };
    const parsed = commandSchema.safeParse(candidate);
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0];
        if (typeof field === "string" && field in values)
          setFieldError(field as keyof Fields, { message: issue.message });
        else setBodyError(issue.message);
      }
      const first = parsed.error.issues[0]?.path[0];
      if (typeof first === "string" && first in values)
        setFocus(first as keyof Fields);
      return;
    }
    setReview(parsed.data);
  }

  async function confirm() {
    if (!review || submitting.current) return;
    submitting.current = true;
    setPending(true);
    setError(null);
    try {
      const result = await create.mutateAsync({
        command: review,
        key: attempt.current.keyFor(review),
      });
      attempt.current.clear();
      setAccepted(result);
      setReview(null);
      reset();
      setAsset(null);
      setIncludeNote(false);
      setIncludeReference(false);
      create.reset();
    } catch (failure) {
      setError(failure);
    } finally {
      submitting.current = false;
      setPending(false);
    }
  }

  const field = (name: keyof Fields, label: string, placeholder?: string) => (
    <div key={name}>
      <label htmlFor={`entry-${name}`} className="text-sm font-medium">
        {label}
      </label>
      <Input
        id={`entry-${name}`}
        {...register(name)}
        placeholder={placeholder}
        aria-invalid={!!errors[name]}
        aria-describedby={errors[name] ? `entry-${name}-error` : undefined}
      />
      {errors[name] && (
        <p
          role="alert"
          id={`entry-${name}-error`}
          className="text-sm text-red-700"
        >
          {errors[name]?.message}
        </p>
      )}
    </div>
  );

  return (
    <section
      aria-labelledby="entry-title"
      className="space-y-4 rounded-lg border bg-white p-5"
    >
      <h2 id="entry-title" className="text-xl font-semibold">
        Record a Transaction
      </h2>
      {accepted ? (
        <div className="space-y-4">
          <h3
            tabIndex={-1}
            ref={focusHeading}
            role="status"
            className="font-semibold"
          >
            Transaction accepted
          </h3>
          <p>Immutable record: {accepted.id}</p>
          <TransactionFacts transaction={accepted} />
          <Button
            type="button"
            onClick={() => {
              setAccepted(null);
              setKind("BUY");
            }}
          >
            Record another entry
          </Button>
        </div>
      ) : (
        <>
          {/* Keep draft fields mounted during review so returning to editing preserves exact input. */}
          <form
            hidden={review !== null}
            onSubmit={handleSubmit(prepare)}
            noValidate
            className="space-y-4"
          >
            <label htmlFor="entry-kind" className="block text-sm font-medium">
              Transaction kind
            </label>
            <select
              id="entry-kind"
              ref={kindInput}
              className="rounded border p-2"
              value={kind}
              onChange={(event) => {
                unregister(["quantity", "unitPrice", "fee", "amount"]);
                setKind(event.target.value as Kind);
                setAsset(null);
                setAssetError(null);
                clearErrors();
              }}
            >
              {createKinds.map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
            <p>
              Currency: USD. Entries become immutable when accepted. No balances
              or projections are calculated here.
            </p>
            {needsAsset && (
              <TransactionAssetPicker
                selected={asset}
                validationError={assetError}
                onSelect={(next) => {
                  setAsset(next);
                  setAssetError(null);
                }}
              />
            )}
            {trade ? (
              <fieldset className="grid gap-3 sm:grid-cols-3">
                <legend>Trade facts</legend>
                {field("quantity", "Quantity")}
                {field("unitPrice", "Unit price (USD)")}
                {field("fee", "Fee (USD, optional; defaults to zero)")}
              </fieldset>
            ) : (
              field("amount", "Amount (USD)")
            )}
            {field(
              "effectiveAt",
              "Effective time (UTC)",
              "2026-01-02T15:30:00Z",
            )}
            <p className="text-sm text-slate-600">
              UTC only, with Z suffix and up to six fractional digits. Past
              entries are allowed, subject to server ledger validation.
            </p>
            <label className="block">
              <input
                type="checkbox"
                checked={includeNote}
                onChange={(event) => setIncludeNote(event.target.checked)}
              />{" "}
              Include note
            </label>
            {includeNote &&
              field("note", "Note (optional, up to 2,000 characters)")}
            <label className="block">
              <input
                type="checkbox"
                checked={includeReference}
                onChange={(event) => setIncludeReference(event.target.checked)}
              />{" "}
              Include external reference
            </label>
            {includeReference &&
              field(
                "externalReference",
                "External reference (optional, up to 256 characters)",
              )}
            <p className="text-sm text-slate-600">
              Included text is preserved exactly, including empty text and
              whitespace.
            </p>
            {bodyError && <p role="alert">{bodyError}</p>}
            <Button type="submit">Review entry</Button>
          </form>
          {review && (
            <div className="space-y-4">
              <h3 tabIndex={-1} ref={focusHeading} className="font-semibold">
                Review entry before submitting
              </h3>
              {needsAsset && asset && (
                <p>
                  {asset.symbol} — {asset.name} · {asset.assetType} ·{" "}
                  {asset.exchange}
                </p>
              )}
              <TransactionFacts transaction={review} />
              <p className="text-sm">
                Confirm these exact facts. The server decides eligibility and
                ledger validity.
              </p>
              {error !== null && (
                <>
                  <TransactionError error={error} />
                  <p>
                    Acceptance may be uncertain. Retry unchanged to reuse this
                    attempt. Before changing facts or leaving, check history;
                    this attempt is held only in memory.
                  </p>
                </>
              )}
              <div className="flex gap-3">
                <Button
                  type="button"
                  disabled={pending}
                  onClick={() => void confirm()}
                >
                  {pending
                    ? "Submitting…"
                    : error !== null
                      ? "Retry same entry"
                      : "Confirm submission"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={pending}
                  onClick={() => {
                    setReview(null);
                    setError(null);
                  }}
                >
                  Return to entry
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </section>
  );
}
