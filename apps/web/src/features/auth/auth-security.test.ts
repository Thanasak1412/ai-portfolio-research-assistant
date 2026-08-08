import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const featureDirectory = path.dirname(fileURLToPath(import.meta.url));
const productionFiles = [
  "api/auth-api.ts",
  "model/auth-session-provider.tsx",
  "model/auth-session.ts",
  "components/credentials-form.tsx",
];

describe("Authentication browser-storage boundary", () => {
  it("does not persist tokens or session data through browser storage APIs", async () => {
    const source = await Promise.all(
      productionFiles.map((file) =>
        readFile(path.join(featureDirectory, file), "utf8"),
      ),
    );
    const productionSource = source.join("\n");
    for (const forbidden of [
      "localStorage",
      "sessionStorage",
      "indexedDB",
      "document.cookie",
      "CacheStorage",
    ]) {
      expect(productionSource).not.toContain(forbidden);
    }
    expect(productionSource).not.toContain("http://localhost:8080");
    expect(productionSource).not.toContain("jwt-decode");
  });
});
