import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import AssetsPage from "./page";

vi.mock("@/features/asset/components/asset-discovery-screen", () => ({
  AssetDiscoveryScreen: () => <p>Asset discovery</p>,
}));

describe("AssetsPage", () => {
  it("mounts the client-side Asset discovery screen", () => {
    render(<AssetsPage />);
    expect(screen.getByText("Asset discovery")).toBeInTheDocument();
  });
});
