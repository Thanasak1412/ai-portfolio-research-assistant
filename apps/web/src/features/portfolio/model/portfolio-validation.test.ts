import { describe, expect, it } from "vitest";

import {
  portfolioNameFormSchema,
  trimPortfolioNameEdges,
} from "@/features/portfolio/model/portfolio-validation";

describe("Portfolio name validation", () => {
  it("trims only approved ASCII edge whitespace for usability validation", () => {
    expect(trimPortfolioNameEdges("\t Growth \r")).toBe("Growth");
    expect(trimPortfolioNameEdges("\u00A0Growth\u00A0")).toBe(
      "\u00A0Growth\u00A0",
    );
    expect(portfolioNameFormSchema.safeParse({ name: "\t \r" }).success).toBe(
      false,
    );
  });

  it("counts Unicode code points and preserves valid internal whitespace", () => {
    expect(
      portfolioNameFormSchema.safeParse({ name: "Growth  Fund" }).success,
    ).toBe(true);
    expect(
      portfolioNameFormSchema.safeParse({ name: "😀".repeat(200) }).success,
    ).toBe(true);
    expect(
      portfolioNameFormSchema.safeParse({ name: "😀".repeat(201) }).success,
    ).toBe(false);
    expect(
      portfolioNameFormSchema.safeParse({
        name: ` ${"😀".repeat(200)} `,
      }).success,
    ).toBe(true);
  });
});
