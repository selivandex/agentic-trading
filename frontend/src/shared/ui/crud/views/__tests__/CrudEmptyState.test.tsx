/** @format */

import { render, screen, fireEvent } from "@testing-library/react";
import { CrudEmptyState } from "../CrudEmptyState";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

interface TestEntity extends CrudEntity {
  name: string;
}

describe("CrudEmptyState", () => {
  const mockOnNew = jest.fn();

  const mockConfig: Partial<CrudConfig<TestEntity>> = {
    resourceName: "Strategy",
    resourceNamePlural: "Strategies",
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should render empty state title", () => {
    render(
      <CrudEmptyState
        config={mockConfig as CrudConfig<TestEntity>}
        onNew={mockOnNew}
      />
    );

    expect(screen.getByText("No strategies")).toBeInTheDocument();
  });

  it("should render default empty state message", () => {
    render(
      <CrudEmptyState
        config={mockConfig as CrudConfig<TestEntity>}
        onNew={mockOnNew}
      />
    );

    expect(
      screen.getByText(/Get started by creating a new strategy/i)
    ).toBeInTheDocument();
  });

  it("should render custom empty state message if provided", () => {
    const configWithCustomMessage: Partial<CrudConfig<TestEntity>> = {
      ...mockConfig,
      emptyStateMessage: "Custom empty message",
    };

    render(
      <CrudEmptyState
        config={configWithCustomMessage as CrudConfig<TestEntity>}
        onNew={mockOnNew}
      />
    );

    expect(screen.getByText("Custom empty message")).toBeInTheDocument();
  });

  it("should render new button", () => {
    render(
      <CrudEmptyState
        config={mockConfig as CrudConfig<TestEntity>}
        onNew={mockOnNew}
      />
    );

    expect(screen.getByText("New Strategy")).toBeInTheDocument();
  });

  it("should call onNew when button clicked", () => {
    render(
      <CrudEmptyState
        config={mockConfig as CrudConfig<TestEntity>}
        onNew={mockOnNew}
      />
    );

    const newButton = screen.getByText("New Strategy");
    fireEvent.click(newButton);

    expect(mockOnNew).toHaveBeenCalled();
  });
});

