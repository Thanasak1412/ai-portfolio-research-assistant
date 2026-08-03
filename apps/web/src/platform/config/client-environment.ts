import { z } from "zod";

const clientEnvironmentSchema = z.object({
  NEXT_PUBLIC_API_BASE_URL: z.url(),
});

export function parseClientEnvironment(
  apiBaseURL: string | undefined,
  nodeEnvironment: string | undefined,
) {
  return clientEnvironmentSchema.parse({
    NEXT_PUBLIC_API_BASE_URL:
      apiBaseURL ??
      (nodeEnvironment === "production"
        ? undefined
        : "http://localhost:8080/api/v1"),
  });
}

export const clientEnvironment = parseClientEnvironment(
  process.env.NEXT_PUBLIC_API_BASE_URL,
  process.env.NODE_ENV,
);
