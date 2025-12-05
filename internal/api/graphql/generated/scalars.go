package generated

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/vektah/gqlparser/v2/ast"
)

// Custom scalar marshal/unmarshal methods for executionContext

// UUID Scalars
func (ec *executionContext) _UUID(ctx context.Context, sel ast.SelectionSet, v *uuid.UUID) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return MarshalUUID(*v)
}

func (ec *executionContext) unmarshalInputUUID(ctx context.Context, v any) (uuid.UUID, error) {
	return UnmarshalUUID(v)
}

// Time Scalars
func (ec *executionContext) _Time(ctx context.Context, sel ast.SelectionSet, v *time.Time) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return MarshalTime(*v)
}

func (ec *executionContext) unmarshalInputTime(ctx context.Context, v any) (time.Time, error) {
	return UnmarshalTime(v)
}

// Decimal Scalars
func (ec *executionContext) _Decimal(ctx context.Context, sel ast.SelectionSet, v *decimal.Decimal) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return MarshalDecimal(*v)
}

func (ec *executionContext) unmarshalInputDecimal(ctx context.Context, v any) (decimal.Decimal, error) {
	return UnmarshalDecimal(v)
}

// JSONObject Scalars
func (ec *executionContext) _JSONObject(ctx context.Context, sel ast.SelectionSet, v map[string]any) graphql.Marshaler {
	return MarshalJSONObject(v)
}

func (ec *executionContext) unmarshalInputJSONObject(ctx context.Context, v any) (map[string]any, error) {
	return UnmarshalJSONObject(v)
}

// Standalone marshal/unmarshal functions

// MarshalUUID marshals UUID to GraphQL
func MarshalUUID(u uuid.UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, fmt.Sprintf(`"%s"`, u.String()))
	})
}

// UnmarshalUUID unmarshals UUID from GraphQL
func UnmarshalUUID(v any) (uuid.UUID, error) {
	switch v := v.(type) {
	case string:
		return uuid.Parse(v)
	case []byte:
		return uuid.Parse(string(v))
	default:
		return uuid.Nil, errors.New("invalid UUID format")
	}
}

// MarshalTime marshals time.Time to GraphQL (RFC3339)
func MarshalTime(t time.Time) graphql.Marshaler {
	if t.IsZero() {
		return graphql.Null
	}
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, fmt.Sprintf(`"%s"`, t.Format(time.RFC3339)))
	})
}

// UnmarshalTime unmarshals time.Time from GraphQL
func UnmarshalTime(v any) (time.Time, error) {
	switch v := v.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	case []byte:
		return time.Parse(time.RFC3339, string(v))
	default:
		return time.Time{}, errors.New("invalid time format, expected RFC3339")
	}
}

// MarshalDecimal marshals decimal.Decimal to GraphQL
func MarshalDecimal(d decimal.Decimal) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, fmt.Sprintf(`"%s"`, d.String()))
	})
}

// UnmarshalDecimal unmarshals decimal.Decimal from GraphQL
func UnmarshalDecimal(v any) (decimal.Decimal, error) {
	switch v := v.(type) {
	case string:
		return decimal.NewFromString(v)
	case int:
		return decimal.NewFromInt(int64(v)), nil
	case int32:
		return decimal.NewFromInt32(v), nil
	case int64:
		return decimal.NewFromInt(v), nil
	case float64:
		return decimal.NewFromFloat(v), nil
	case []byte:
		return decimal.NewFromString(string(v))
	default:
		return decimal.Zero, fmt.Errorf("invalid decimal format: %T", v)
	}
}

// MarshalJSONObject marshals generic JSON object to GraphQL
func MarshalJSONObject(v any) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		data, err := json.Marshal(v)
		if err != nil {
			_, _ = io.WriteString(w, "null")
			return
		}
		_, _ = w.Write(data)
	})
}

// UnmarshalJSONObject unmarshals generic JSON object from GraphQL
func UnmarshalJSONObject(v any) (map[string]any, error) {
	switch v := v.(type) {
	case map[string]any:
		return v, nil
	case string:
		var result map[string]any
		if err := json.Unmarshal([]byte(v), &result); err != nil {
			return nil, err
		}
		return result, nil
	case []byte:
		var result map[string]any
		if err := json.Unmarshal(v, &result); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, fmt.Errorf("invalid JSONObject format: %T", v)
	}
}
