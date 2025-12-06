/** @format */

import { render, screen, fireEvent } from "@testing-library/react";
import { CrudPagination } from "../CrudPagination";
import type { PageInfo } from "@/shared/lib/crud/types";

describe("CrudPagination", () => {
  const mockOnLoadMore = jest.fn();

  const mockPageInfo: PageInfo = {
    hasNextPage: true,
    hasPreviousPage: false,
    startCursor: "cursor1",
    endCursor: "cursor2",
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should render current count and total count", () => {
    render(
      <CrudPagination
        pageInfo={mockPageInfo}
        totalCount={100}
        currentCount={20}
        pageSize={20}
        loading={false}
        onLoadMore={mockOnLoadMore}
      />
    );

    expect(screen.getByText("Showing 20 of 100")).toBeInTheDocument();
  });

  it("should render load more button when hasNextPage is true", () => {
    render(
      <CrudPagination
        pageInfo={mockPageInfo}
        totalCount={100}
        currentCount={20}
        pageSize={20}
        loading={false}
        onLoadMore={mockOnLoadMore}
      />
    );

    expect(screen.getByText("Load More")).toBeInTheDocument();
  });

  it("should not render load more button when hasNextPage is false", () => {
    const pageInfoNoNext: PageInfo = {
      ...mockPageInfo,
      hasNextPage: false,
    };

    render(
      <CrudPagination
        pageInfo={pageInfoNoNext}
        totalCount={100}
        currentCount={100}
        pageSize={20}
        loading={false}
        onLoadMore={mockOnLoadMore}
      />
    );

    expect(screen.queryByText("Load More")).not.toBeInTheDocument();
  });

  it("should call onLoadMore when button clicked", () => {
    render(
      <CrudPagination
        pageInfo={mockPageInfo}
        totalCount={100}
        currentCount={20}
        pageSize={20}
        loading={false}
        onLoadMore={mockOnLoadMore}
      />
    );

    const loadMoreButton = screen.getByText("Load More");
    fireEvent.click(loadMoreButton);

    expect(mockOnLoadMore).toHaveBeenCalled();
  });

  it("should disable load more button when loading", () => {
    render(
      <CrudPagination
        pageInfo={mockPageInfo}
        totalCount={100}
        currentCount={20}
        pageSize={20}
        loading={true}
        onLoadMore={mockOnLoadMore}
      />
    );

    const loadMoreButton = screen.getByText("Loading...").closest("button");
    expect(loadMoreButton).toBeDisabled();
  });

  it("should show loading text when loading", () => {
    render(
      <CrudPagination
        pageInfo={mockPageInfo}
        totalCount={100}
        currentCount={20}
        pageSize={20}
        loading={true}
        onLoadMore={mockOnLoadMore}
      />
    );

    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("should not render count if totalCount is undefined", () => {
    render(
      <CrudPagination
        pageInfo={mockPageInfo}
        totalCount={undefined}
        currentCount={20}
        pageSize={20}
        loading={false}
        onLoadMore={mockOnLoadMore}
      />
    );

    expect(screen.queryByText(/Showing/)).not.toBeInTheDocument();
  });
});

