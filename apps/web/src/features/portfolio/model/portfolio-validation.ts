import { z } from "zod";

const asciiEdgeWhitespace =
  /^[\x20\x09\x0A\x0D\x0C\x0B]+|[\x20\x09\x0A\x0D\x0C\x0B]+$/g;

export function trimPortfolioNameEdges(value: string): string {
  return value.replace(asciiEdgeWhitespace, "");
}

export const portfolioNameSchema = z
  .string()
  .refine((value) => trimPortfolioNameEdges(value) !== "", {
    message: "Portfolio name is required.",
  })
  .refine((value) => Array.from(value).length <= 200, {
    message: "Portfolio name must be 200 characters or fewer.",
  });

export const portfolioNameFormSchema = z.object({ name: portfolioNameSchema });
export type PortfolioNameFormValues = z.infer<typeof portfolioNameFormSchema>;
