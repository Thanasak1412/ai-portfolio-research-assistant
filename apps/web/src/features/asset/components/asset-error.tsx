import { ApiError } from "@/platform/api/api-error";

export function assetErrorMessage(error: unknown): string {
  if (!(error instanceof ApiError)) {
    return "Asset data is temporarily unavailable. Please try again.";
  }
  if (error.code === "INVALID_REQUEST") {
    return "The current Asset discovery request is invalid.";
  }
  return "Asset data is temporarily unavailable. Please try again.";
}

export function AssetError({ error }: Readonly<{ error: unknown }>) {
  const correlationId =
    error instanceof ApiError ? error.correlationId : undefined;
  return (
    <div
      role="alert"
      className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-800"
    >
      <p>{assetErrorMessage(error)}</p>
      {correlationId ? (
        <p className="mt-1 text-xs">Reference: {correlationId}</p>
      ) : null}
    </div>
  );
}
