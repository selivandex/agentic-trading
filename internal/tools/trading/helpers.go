package trading

import (
	"fmt"

	"github.com/google/uuid"
)

func parseUUIDArg(raw interface{}, name string) (uuid.UUID, error) {
	switch v := raw.(type) {
	case string:
		if v == "" {
			return uuid.Nil, fmt.Errorf("%s is required", name)
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, fmt.Errorf("%s is invalid: %w", name, err)
		}
		return id, nil
	case uuid.UUID:
		if v == uuid.Nil {
			return uuid.Nil, fmt.Errorf("%s is required", name)
		}
		return v, nil
	default:
		return uuid.Nil, fmt.Errorf("%s is required", name)
	}
}
