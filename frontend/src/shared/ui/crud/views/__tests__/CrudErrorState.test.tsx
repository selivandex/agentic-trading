/** @format */

import { render, screen, fireEvent } from "@testing-library/react";
import { CrudErrorState } from "../CrudErrorState";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

interface TestEntity extends CrudEntity {
  name: string;
}

describe("CrudErrorState", () => {
  const mockOnRetry = jest.fn();

  const mockConfig: Partial<CrudConfig<TestEntity>> = {
    resourceName: "Strategy",
  };

  const mockError = new Error("Network error");

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should render error title", () => {
    render(
      <CrudErrorState
        config={mockConfig as CrudConfig<TestEntity>}
        error={mockError}
        onRetry={mockOnRetry}
      />
    );

    expect(screen.getByText("Error loading data")).toBeInTheDocument();
  });

  it("should render error message", () => {
    render(
      <CrudErrorState
        config={mockConfig as CrudConfig<TestEntity>}
        error={mockError}
        onRetry={mockOnRetry}
      />
    );

    expect(screen.getByText("Network error")).toBeInTheDocument();
  });

  it("should render custom error message if provided", () => {
    const configWithCustomError: Partial<CrudConfig<TestEntity>> = {
      ...mockConfig,
      errorMessage: "Custom error message",
    };

    render(
      <CrudErrorState
        config={configWithCustomError as CrudConfig<TestEntity>}
        error={mockError}
        onRetry={mockOnRetry}
      />
    );

    expect(screen.getByText("Custom error message")).toBeInTheDocument();
  });

  it("should render retry button", () => {
    render(
      <CrudErrorState
        config={mockConfig as CrudConfig<TestEntity>}
        error={mockError}
        onRetry={mockOnRetry}
      />
    );

    expect(screen.getByText("Try Again")).toBeInTheDocument();
  });

  it("should call onRetry when button clicked", () => {
    render(
      <CrudErrorState
        config={mockConfig as CrudConfig<TestEntity>}
        error={mockError}
        onRetry={mockOnRetry}
      />
    );

    const retryButton = screen.getByText("Try Again");
    fireEvent.click(retryButton);

    expect(mockOnRetry).toHaveBeenCalled();
  });

  it("should render default message if no error provided", () => {
    render(
      <CrudErrorState
        config={mockConfig as CrudConfig<TestEntity>}
        error={undefined}
        onRetry={mockOnRetry}
      />
    );

    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });
});

