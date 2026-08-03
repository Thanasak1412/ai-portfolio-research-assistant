import { describe, expect, it } from "vitest";

import { parseClientEnvironment } from "./client-environment";

describe("parseClientEnvironment", () => {
  it("uses the local API only outside production", () => {
    expect(
      parseClientEnvironment(undefined, "development").NEXT_PUBLIC_API_BASE_URL,
    ).toBe("http://localhost:8080/api/v1");
  });

  it("requires an explicit production API URL", () => {
    expect(() => parseClientEnvironment(undefined, "production")).toThrow();
  });

  it("rejects an invalid API URL", () => {
    expect(() => parseClientEnvironment("not-a-url", "production")).toThrow();
  });
});
