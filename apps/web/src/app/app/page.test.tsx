import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import ProtectedAppPage from "./page";

describe("ProtectedAppPage", () => {
  it("offers neutral Portfolio and Asset discovery entry points", () => {
    render(<ProtectedAppPage />);
    expect(
      screen.getByRole("link", { name: "View Portfolios" }),
    ).toHaveAttribute("href", "/app/portfolios");
    expect(
      screen.getByRole("link", { name: "Discover Assets" }),
    ).toHaveAttribute("href", "/app/assets");
    expect(
      screen.queryByText(/Market Value|Holdings|Price/),
    ).not.toBeInTheDocument();
  });
});
