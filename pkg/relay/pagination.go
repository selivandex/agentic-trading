// Package relay provides utilities for implementing Relay-style cursor pagination
package relay

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// PageInfo represents pagination information following the Relay spec
type PageInfo struct {
	HasNextPage     bool
	HasPreviousPage bool
	StartCursor     *string
	EndCursor       *string
}

// Edge represents a single edge in a connection
type Edge[T any] struct {
	Node   T
	Cursor string
}

// Connection represents a paginated collection following the Relay spec
type Connection[T any] struct {
	Edges      []Edge[T]
	PageInfo   PageInfo
	TotalCount int
}

// PaginationParams represents the pagination parameters
type PaginationParams struct {
	First  *int
	After  *string
	Last   *int
	Before *string
}

// Validate validates the pagination parameters
func (p PaginationParams) Validate() error {
	if p.First != nil && p.Last != nil {
		return fmt.Errorf("cannot provide both 'first' and 'last'")
	}

	if p.First != nil && *p.First < 0 {
		return fmt.Errorf("'first' must be non-negative")
	}

	if p.Last != nil && *p.Last < 0 {
		return fmt.Errorf("'last' must be non-negative")
	}

	if p.After != nil && p.Before != nil {
		return fmt.Errorf("cannot provide both 'after' and 'before'")
	}

	return nil
}

// GetLimit returns the effective limit for the query
func (p PaginationParams) GetLimit() int {
	if p.First != nil {
		return *p.First
	}
	if p.Last != nil {
		return *p.Last
	}
	return 20 // Default limit
}

// IsForward returns true if this is forward pagination
func (p PaginationParams) IsForward() bool {
	return p.First != nil || (p.First == nil && p.Last == nil)
}

// EncodeCursor encodes an offset into a cursor string
func EncodeCursor(offset int) string {
	str := fmt.Sprintf("cursor:%d", offset)
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// DecodeCursor decodes a cursor string into an offset
func DecodeCursor(cursor string) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor: %w", err)
	}

	str := string(decoded)
	if !strings.HasPrefix(str, "cursor:") {
		return 0, fmt.Errorf("invalid cursor format")
	}

	offsetStr := strings.TrimPrefix(str, "cursor:")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor offset: %w", err)
	}

	return offset, nil
}

// NewConnection creates a new connection from items and pagination params
func NewConnection[T any](
	items []T,
	totalCount int,
	params PaginationParams,
	startOffset int,
) (*Connection[T], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	edges := make([]Edge[T], len(items))
	for i, item := range items {
		edges[i] = Edge[T]{
			Node:   item,
			Cursor: EncodeCursor(startOffset + i),
		}
	}

	pageInfo := PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
	}

	if len(edges) > 0 {
		startCursor := edges[0].Cursor
		endCursor := edges[len(edges)-1].Cursor
		pageInfo.StartCursor = &startCursor
		pageInfo.EndCursor = &endCursor

		// Calculate if there are more pages
		if params.IsForward() {
			// Forward pagination
			pageInfo.HasPreviousPage = startOffset > 0
			pageInfo.HasNextPage = startOffset+len(items) < totalCount
		} else {
			// Backward pagination
			pageInfo.HasPreviousPage = startOffset > 0
			pageInfo.HasNextPage = startOffset+len(items) < totalCount
		}
	}

	return &Connection[T]{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: totalCount,
	}, nil
}

// CalculateOffsetLimit calculates the offset and limit for a database query
// based on the pagination parameters
func CalculateOffsetLimit(params PaginationParams, totalCount int) (offset, limit int, err error) {
	if err := params.Validate(); err != nil {
		return 0, 0, err
	}

	limit = params.GetLimit()
	offset = 0

	if params.After != nil {
		afterOffset, err := DecodeCursor(*params.After)
		if err != nil {
			return 0, 0, err
		}
		offset = afterOffset + 1
	}

	if params.Before != nil {
		beforeOffset, err := DecodeCursor(*params.Before)
		if err != nil {
			return 0, 0, err
		}
		offset = beforeOffset - limit
		if offset < 0 {
			offset = 0
		}
	}

	if params.Last != nil {
		// For backward pagination, we need to calculate from the end
		offset = totalCount - limit
		if offset < 0 {
			offset = 0
		}

		if params.Before != nil {
			beforeOffset, err := DecodeCursor(*params.Before)
			if err != nil {
				return 0, 0, err
			}
			offset = beforeOffset - limit
			if offset < 0 {
				offset = 0
			}
		}
	}

	return offset, limit, nil
}
