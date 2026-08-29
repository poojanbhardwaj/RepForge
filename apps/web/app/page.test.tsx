import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import HomePage from "./page";

describe("HomePage", () => {
  it("provides a semantic heading and honest bootstrap status", () => {
    render(<HomePage />);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Know what to do today");
    expect(screen.getByRole("status")).toHaveTextContent(
      "No public accounts or subscriptions are enabled",
    );
    expect(screen.getByRole("navigation", { name: "Primary navigation" })).toBeInTheDocument();
  });
});
