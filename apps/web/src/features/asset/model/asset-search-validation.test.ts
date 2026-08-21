import { describe, expect, it } from "vitest";

import {
  assetSearchValidationMessage,
  maximumAssetSearchCodePoints,
} from "@/features/asset/model/asset-search-validation";

describe("Asset search validation", () => {
  it("counts Unicode code points without changing search semantics", () => {
    expect(assetSearchValidationMessage("  BTC  ")).toBeNull();
    expect(
      assetSearchValidationMessage("😀".repeat(maximumAssetSearchCodePoints)),
    ).toBeNull();
    expect(
      assetSearchValidationMessage(
        "😀".repeat(maximumAssetSearchCodePoints + 1),
      ),
    ).toBe("Search must be 100 characters or fewer.");
  });
});
