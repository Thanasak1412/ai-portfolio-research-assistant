import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const featureDirectory = path.dirname(fileURLToPath(import.meta.url));
const productionFiles = [
  "api/auth-api.ts",
  "model/auth-session-provider.tsx",
  "model/auth-browser-coordinator.ts",
  "model/auth-session.ts",
  "components/credentials-form.tsx",
  "components/logout-button.tsx",
  "components/require-authenticated.tsx",
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
    expect(productionSource).not.toContain("console.");
    expect(productionSource).not.toContain("setInterval(");
  });

  it("keeps browser coordination payloads free of Authentication material", async () => {
    const source = await readFile(
      path.join(featureDirectory, "model/auth-browser-coordinator.ts"),
      "utf8",
    );
    for (const forbidden of [
      "accessToken",
      "refreshToken",
      "Authorization",
      "Bearer",
      "pra_rt_v1",
      "email",
    ]) {
      expect(source).not.toContain(forbidden);
    }
  });
});
