package scalars

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MarshalUUID marshals UUID to GraphQL
func MarshalUUID(u uuid.UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, fmt.Sprintf(`"%s"`, u.String()))
	})
}

// UnmarshalUUID unmarshals UUID from GraphQL
func UnmarshalUUID(v interface{}) (uuid.UUID, error) {
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
func UnmarshalTime(v interface{}) (time.Time, error) {
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
func UnmarshalDecimal(v interface{}) (decimal.Decimal, error) {
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
func MarshalJSONObject(v interface{}) graphql.Marshaler {
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
func UnmarshalJSONObject(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case map[string]interface{}:
		return v, nil
	case string:
		var result interface{}
		if err := json.Unmarshal([]byte(v), &result); err != nil {
			return nil, err
		}
		return result, nil
	case []byte:
		var result interface{}
		if err := json.Unmarshal(v, &result); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return v, nil
	}
}
