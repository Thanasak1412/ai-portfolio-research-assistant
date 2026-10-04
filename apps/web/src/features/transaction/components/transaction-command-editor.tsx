"use client";

import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { Asset } from "@/features/asset/api/asset-api";
import type { TransactionCommand } from "@/features/transaction/api/transaction-api";
import {
  commandSchema,
  correctionCommandSchema,
  createKinds,
  isEligibleAsset,
} from "@/features/transaction/model/transaction-validation";
import { TransactionAssetPicker } from "@/features/transaction/components/transaction-asset-picker";

type Fields = {
  effectiveAt: string;
  quantity?: string;
  unitPrice?: string;
  fee?: string;
  amount?: string;
  note?: string;
  externalReference?: string;
};

// One field matrix for create and complete replacement. Remains mounted (hidden)
// during review so returning to the editor preserves the exact draft strings.
export function TransactionCommandEditor({
  mode = "entry",
  hidden,
  onReview,
  focusOnMount = false,
}: Readonly<{
  mode?: "entry" | "replacement";
  hidden: boolean;
  focusOnMount?: boolean;
  onReview: (command: TransactionCommand, asset: Asset | null) => void;
}>) {
  const [kind, setKind] = useState<TransactionCommand["kind"]>("BUY");
  const [asset, setAsset] = useState<Asset | null>(null);
  const [includeNote, setIncludeNote] = useState(false);
  const [includeReference, setIncludeReference] = useState(false);
  const [assetError, setAssetError] = useState<string | null>(null);
  const [bodyError, setBodyError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    setError,
    clearErrors,
    unregister,
    setFocus,
    formState: { errors },
  } = useForm<Fields>({
    shouldUnregister: true,
    defaultValues: { effectiveAt: "" },
  });
  const kindInput = useRef<HTMLSelectElement>(null);
  const wasHidden = useRef(hidden);
  useEffect(() => {
    if (!hidden && (wasHidden.current || focusOnMount))
      kindInput.current?.focus();
    wasHidden.current = hidden;
  }, [hidden, focusOnMount]);
  const trade = kind === "BUY" || kind === "SELL";
  const needsAsset = trade || kind === "DIVIDEND";

  function prepare(values: Fields) {
    clearErrors();
    setAssetError(null);
    setBodyError(null);
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
          setError(field as keyof Fields, { message: issue.message });
        else setBodyError(issue.message);
      }
      const first = parsed.error.issues[0]?.path[0];
      if (typeof first === "string" && first in values)
        setFocus(first as keyof Fields);
      return;
    }
    if (mode === "replacement") {
      const correction = correctionCommandSchema.safeParse({
        replacement: parsed.data,
      });
      if (!correction.success) {
        setBodyError(correction.error.issues[0].message);
        return;
      }
    }
    onReview(parsed.data, asset);
  }
  const field = (name: keyof Fields, label: string, placeholder?: string) => (
    <div key={name}>
      <label htmlFor={`${mode}-${name}`} className="text-sm font-medium">
        {label}
      </label>
      <Input
        id={`${mode}-${name}`}
        {...register(name)}
        placeholder={placeholder}
        aria-invalid={!!errors[name]}
        aria-describedby={errors[name] ? `${mode}-${name}-error` : undefined}
      />
      {errors[name] && (
        <p
          role="alert"
          id={`${mode}-${name}-error`}
          className="text-sm text-red-700"
        >
          {errors[name]?.message}
        </p>
      )}
    </div>
  );
  return (
    <form
      hidden={hidden}
      onSubmit={handleSubmit(prepare)}
      noValidate
      className="space-y-4"
      aria-label={
        mode === "entry" ? "Transaction entry" : "Complete replacement"
      }
    >
      <label htmlFor={`${mode}-kind`} className="block text-sm font-medium">
        {mode === "entry" ? "Transaction kind" : "Replacement kind"}
      </label>
      <select
        id={`${mode}-kind`}
        ref={kindInput}
        className="rounded border p-2"
        value={kind}
        onChange={(event) => {
          unregister(["quantity", "unitPrice", "fee", "amount"]);
          setKind(event.target.value as TransactionCommand["kind"]);
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
        Currency: USD. Entries become immutable when accepted. No balances or
        projections are calculated here.
      </p>
      {needsAsset && (
        <TransactionAssetPicker
          idPrefix={mode}
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
      {field("effectiveAt", "Effective time (UTC)", "2026-01-02T15:30:00Z")}
      <p className="text-sm text-slate-600">
        UTC only, with Z suffix and up to six fractional digits. Past entries
        are allowed, subject to server ledger validation.
      </p>
      <label className="block">
        <input
          type="checkbox"
          checked={includeNote}
          onChange={(event) => setIncludeNote(event.target.checked)}
        />{" "}
        Include note
      </label>
      {includeNote && field("note", "Note (optional, up to 2,000 characters)")}
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
        Included text is preserved exactly, including empty text and whitespace.
      </p>
      {bodyError && <p role="alert">{bodyError}</p>}
      <Button type="submit">
        {mode === "entry" ? "Review entry" : "Review correction"}
      </Button>
    </form>
  );
}
