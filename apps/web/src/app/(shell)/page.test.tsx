import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import BootstrapPage from "./page";

describe("BootstrapPage", () => {
  it("identifies the application as a foundation without business functionality", () => {
    render(<BootstrapPage />);
    expect(
      screen.getByRole("heading", { name: "Portfolio Research Assistant" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Business functionality begins in later milestones/),
    ).toBeInTheDocument();
  });
});
