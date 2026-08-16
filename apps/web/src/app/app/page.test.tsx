import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import ProtectedAppPage from "./page";

describe("ProtectedAppPage", () => {
  it("offers the neutral Portfolio entry point without premature M2 UI", () => {
    render(<ProtectedAppPage />);
    expect(
      screen.getByRole("link", { name: "View Portfolios" }),
    ).toHaveAttribute("href", "/app/portfolios");
    expect(
      screen.queryByText(/Assets|Market Value|Holdings/),
    ).not.toBeInTheDocument();
  });
});
