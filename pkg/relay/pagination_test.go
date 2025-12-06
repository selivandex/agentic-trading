package relay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	tests := []struct {
		name   string
		offset int
	}{
		{"zero", 0},
		{"positive", 42},
		{"large", 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := EncodeCursor(tt.offset)
			decoded, err := DecodeCursor(cursor)
			require.NoError(t, err)
			assert.Equal(t, tt.offset, decoded)
		})
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{"empty", ""},
		{"invalid base64", "!!!invalid!!!"},
		{"invalid format", "aGVsbG8="},         // "hello" in base64
		{"invalid offset", "Y3Vyc29yOmFiYw=="}, // "cursor:abc" in base64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeCursor(tt.cursor)
			assert.Error(t, err)
		})
	}
}

func TestPaginationParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  PaginationParams
		wantErr bool
	}{
		{
			name:    "valid forward",
			params:  PaginationParams{First: intPtr(10)},
			wantErr: false,
		},
		{
			name:    "valid backward",
			params:  PaginationParams{Last: intPtr(10)},
			wantErr: false,
		},
		{
			name:    "valid with cursors",
			params:  PaginationParams{First: intPtr(10), After: strPtr("cursor")},
			wantErr: false,
		},
		{
			name:    "both first and last",
			params:  PaginationParams{First: intPtr(10), Last: intPtr(10)},
			wantErr: true,
		},
		{
			name:    "negative first",
			params:  PaginationParams{First: intPtr(-1)},
			wantErr: true,
		},
		{
			name:    "negative last",
			params:  PaginationParams{Last: intPtr(-1)},
			wantErr: true,
		},
		{
			name:    "both after and before",
			params:  PaginationParams{First: intPtr(10), After: strPtr("a"), Before: strPtr("b")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPaginationParams_GetLimit(t *testing.T) {
	tests := []struct {
		name      string
		params    PaginationParams
		wantLimit int
	}{
		{
			name:      "with first",
			params:    PaginationParams{First: intPtr(25)},
			wantLimit: 25,
		},
		{
			name:      "with last",
			params:    PaginationParams{Last: intPtr(15)},
			wantLimit: 15,
		},
		{
			name:      "no params (default)",
			params:    PaginationParams{},
			wantLimit: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := tt.params.GetLimit()
			assert.Equal(t, tt.wantLimit, limit)
		})
	}
}

func TestPaginationParams_IsForward(t *testing.T) {
	tests := []struct {
		name        string
		params      PaginationParams
		wantForward bool
	}{
		{
			name:        "first (forward)",
			params:      PaginationParams{First: intPtr(10)},
			wantForward: true,
		},
		{
			name:        "last (backward)",
			params:      PaginationParams{Last: intPtr(10)},
			wantForward: false,
		},
		{
			name:        "no params (default forward)",
			params:      PaginationParams{},
			wantForward: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isForward := tt.params.IsForward()
			assert.Equal(t, tt.wantForward, isForward)
		})
	}
}

func TestNewConnection(t *testing.T) {
	items := []string{"item1", "item2", "item3"}

	t.Run("forward pagination", func(t *testing.T) {
		conn, err := NewConnection(
			items,
			10, // total count
			PaginationParams{First: intPtr(3)},
			0, // start offset
		)
		require.NoError(t, err)

		assert.Len(t, conn.Edges, 3)
		assert.Equal(t, 10, conn.TotalCount)
		assert.False(t, conn.PageInfo.HasPreviousPage)
		assert.True(t, conn.PageInfo.HasNextPage)
		assert.NotNil(t, conn.PageInfo.StartCursor)
		assert.NotNil(t, conn.PageInfo.EndCursor)
	})

	t.Run("last page", func(t *testing.T) {
		conn, err := NewConnection(
			items,
			8, // total count
			PaginationParams{First: intPtr(3)},
			5, // start offset (items 5, 6, 7)
		)
		require.NoError(t, err)

		assert.Len(t, conn.Edges, 3)
		assert.True(t, conn.PageInfo.HasPreviousPage)
		assert.False(t, conn.PageInfo.HasNextPage)
	})

	t.Run("empty result", func(t *testing.T) {
		conn, err := NewConnection(
			[]string{},
			0,
			PaginationParams{First: intPtr(10)},
			0,
		)
		require.NoError(t, err)

		assert.Empty(t, conn.Edges)
		assert.Equal(t, 0, conn.TotalCount)
		assert.Nil(t, conn.PageInfo.StartCursor)
		assert.Nil(t, conn.PageInfo.EndCursor)
	})
}

func TestCalculateOffsetLimit(t *testing.T) {
	tests := []struct {
		name       string
		params     PaginationParams
		totalCount int
		wantOffset int
		wantLimit  int
		wantErr    bool
	}{
		{
			name:       "first page",
			params:     PaginationParams{First: intPtr(10)},
			totalCount: 100,
			wantOffset: 0,
			wantLimit:  10,
			wantErr:    false,
		},
		{
			name:       "with after cursor",
			params:     PaginationParams{First: intPtr(10), After: strPtr(EncodeCursor(5))},
			totalCount: 100,
			wantOffset: 6,
			wantLimit:  10,
			wantErr:    false,
		},
		{
			name:       "last page with last",
			params:     PaginationParams{Last: intPtr(10)},
			totalCount: 25,
			wantOffset: 15,
			wantLimit:  10,
			wantErr:    false,
		},
		{
			name:       "last smaller than total",
			params:     PaginationParams{Last: intPtr(30)},
			totalCount: 25,
			wantOffset: 0,
			wantLimit:  30,
			wantErr:    false,
		},
		{
			name:       "before cursor",
			params:     PaginationParams{First: intPtr(10), Before: strPtr(EncodeCursor(20))},
			totalCount: 100,
			wantOffset: 10,
			wantLimit:  10,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, limit, err := CalculateOffsetLimit(tt.params, tt.totalCount)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantOffset, offset)
				assert.Equal(t, tt.wantLimit, limit)
			}
		})
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
