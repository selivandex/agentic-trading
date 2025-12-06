/** @format */

import { render, screen, fireEvent } from "@testing-library/react";
import { CrudListHeader } from "../CrudListHeader";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

interface TestEntity extends CrudEntity {
  name: string;
}

describe("CrudListHeader", () => {
  const mockOnSearchChange = jest.fn();
  const mockOnNew = jest.fn();

  const mockConfig: Partial<CrudConfig<TestEntity>> = {
    resourceName: "Strategy",
    resourceNamePlural: "Strategies",
    enableSearch: true,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should render title with entity count", () => {
    render(
      <CrudListHeader
        config={mockConfig as CrudConfig<TestEntity>}
        entitiesCount={5}
        searchQuery=""
        onSearchChange={mockOnSearchChange}
        onNew={mockOnNew}
      />
    );

    expect(screen.getByText("Strategies")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
  });

  it("should render search input when enabled", () => {
    render(
      <CrudListHeader
        config={mockConfig as CrudConfig<TestEntity>}
        entitiesCount={5}
        searchQuery=""
        onSearchChange={mockOnSearchChange}
        onNew={mockOnNew}
      />
    );

    const searchInput = screen.getByPlaceholderText("Search...");
    expect(searchInput).toBeInTheDocument();
  });

  it("should not render search input when disabled", () => {
    const configWithoutSearch: Partial<CrudConfig<TestEntity>> = {
      ...mockConfig,
      enableSearch: false,
    };

    render(
      <CrudListHeader
        config={configWithoutSearch as CrudConfig<TestEntity>}
        entitiesCount={5}
        searchQuery=""
        onSearchChange={mockOnSearchChange}
        onNew={mockOnNew}
      />
    );

    expect(screen.queryByPlaceholderText("Search...")).not.toBeInTheDocument();
  });

  it("should handle search input change", () => {
    render(
      <CrudListHeader
        config={mockConfig as CrudConfig<TestEntity>}
        entitiesCount={5}
        searchQuery=""
        onSearchChange={mockOnSearchChange}
        onNew={mockOnNew}
      />
    );

    const searchInput = screen.getByPlaceholderText("Search...");
    fireEvent.change(searchInput, { target: { value: "test query" } });

    expect(mockOnSearchChange).toHaveBeenCalled();
  });

  it("should render new button", () => {
    render(
      <CrudListHeader
        config={mockConfig as CrudConfig<TestEntity>}
        entitiesCount={5}
        searchQuery=""
        onSearchChange={mockOnSearchChange}
        onNew={mockOnNew}
      />
    );

    const newButton = screen.getByText("New Strategy");
    expect(newButton).toBeInTheDocument();
  });

  it("should call onNew when new button clicked", () => {
    render(
      <CrudListHeader
        config={mockConfig as CrudConfig<TestEntity>}
        entitiesCount={5}
        searchQuery=""
        onSearchChange={mockOnSearchChange}
        onNew={mockOnNew}
      />
    );

    const newButton = screen.getByText("New Strategy");
    fireEvent.click(newButton);

    expect(mockOnNew).toHaveBeenCalled();
  });

  it("should display current search query", () => {
    render(
      <CrudListHeader
        config={mockConfig as CrudConfig<TestEntity>}
        entitiesCount={5}
        searchQuery="my search"
        onSearchChange={mockOnSearchChange}
        onNew={mockOnNew}
      />
    );

    const searchInput = screen.getByPlaceholderText(
      "Search..."
    ) as HTMLInputElement;
    expect(searchInput.value).toBe("my search");
  });
});

