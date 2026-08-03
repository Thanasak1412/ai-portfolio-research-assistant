import { clientEnvironment } from "@/platform/config/client-environment";

export class ApiRequestError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly correlationId: string | null,
  ) {
    super(message);
    this.name = "ApiRequestError";
  }
}

export async function requestApi<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(
    `${clientEnvironment.NEXT_PUBLIC_API_BASE_URL}${path}`,
    {
      ...init,
      headers: { Accept: "application/json", ...init?.headers },
    },
  );

  if (!response.ok) {
    throw new ApiRequestError(
      "The API request failed",
      response.status,
      response.headers.get("x-correlation-id"),
    );
  }

  return (await response.json()) as T;
}
