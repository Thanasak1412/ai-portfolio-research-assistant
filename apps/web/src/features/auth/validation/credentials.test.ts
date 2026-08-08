import { describe, expect, it } from "vitest";
import {
  credentialsSchema,
  normalizedCredentials,
} from "@/features/auth/validation/credentials";

const validCredential = "x".repeat(12);

describe("credentialsSchema", () => {
  it("trims email and accepts no password composition rule", () =>
    expect(
      credentialsSchema.parse({
        email: " person@example.com ",
        password: validCredential,
      }).email,
    ).toBe("person@example.com"));
  it("rejects short and UTF-8-byte oversized passwords", () => {
    expect(
      credentialsSchema.safeParse({
        email: "person@example.com",
        password: "short",
      }).success,
    ).toBe(false);
    expect(
      credentialsSchema.safeParse({
        email: "person@example.com",
        password: "😀".repeat(257),
      }).success,
    ).toBe(false);
  });
  it("normalizes only surrounding email whitespace", () =>
    expect(
      normalizedCredentials({
        email: " person@example.com ",
        password: validCredential,
      }).email,
    ).toBe("person@example.com"));
});
