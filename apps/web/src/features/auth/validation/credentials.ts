import { z } from "zod";

const maximumCredentialBytes = 1_024;
const encoder = new TextEncoder();

export const credentialsSchema = z.object({
  email: z
    .string()
    .trim()
    .min(1, "Email is required")
    .email("Enter a valid email address"),
  password: z
    .string()
    .min(12, "Password must be at least 12 characters")
    .refine(
      (value) => encoder.encode(value).byteLength <= maximumCredentialBytes,
      "Password must not exceed 1,024 bytes",
    ),
});

export type CredentialsFormValues = z.infer<typeof credentialsSchema>;

export function normalizedCredentials(
  values: CredentialsFormValues,
): CredentialsFormValues {
  return { ...values, email: values.email.trim() };
}
