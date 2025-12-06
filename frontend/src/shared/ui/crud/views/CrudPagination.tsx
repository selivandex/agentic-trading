/** @format */

"use client";

import React from "react";
import { Plus } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import type { PageInfo } from "@/shared/lib/crud/types";

/**
 * Pagination controls presentation component
 */
interface CrudPaginationProps {
  pageInfo?: PageInfo;
  totalCount?: number;
  currentCount: number;
  pageSize: number;
  loading: boolean;
  onLoadMore: () => void;
}

export function CrudPagination({
  pageInfo,
  totalCount,
  currentCount,
  loading,
  onLoadMore,
}: CrudPaginationProps) {
  return (
    <div className="flex items-center justify-between mt-4">
      {/* Total count - left side, subtle */}
      <div className="flex-1">
        {totalCount !== undefined && currentCount > 0 && (
          <span className="text-xs text-quaternary">
            Showing {currentCount} of {totalCount}
          </span>
        )}
      </div>

      {/* Load More Button - center */}
      <div className="flex-1 flex justify-center">
        {pageInfo?.hasNextPage && (
          <Button
            size="md"
            color="secondary"
            onClick={onLoadMore}
            iconLeading={Plus}
            isDisabled={loading}
          >
            {loading ? "Loading..." : "Load More"}
          </Button>
        )}
      </div>

      {/* Empty space for balance */}
      <div className="flex-1" />
    </div>
  );
}

