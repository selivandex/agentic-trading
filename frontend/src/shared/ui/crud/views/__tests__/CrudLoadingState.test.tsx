/** @format */

import { render, screen } from "@testing-library/react";
import { CrudLoadingState } from "../CrudLoadingState";

describe("CrudLoadingState", () => {
  it("should render loading skeleton", () => {
    const { container } = render(<CrudLoadingState />);

    // Check for skeleton elements
    const skeletons = container.querySelectorAll(".h-5, .h-10");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("should render multiple skeleton rows", () => {
    const { container } = render(<CrudLoadingState />);

    // Should render 5 skeleton rows based on Array.from({ length: 5 })
    const rows = container.querySelectorAll(".flex.items-center.gap-4.py-4");
    expect(rows.length).toBe(5);
  });
});

