package trading

import (
	"prometheus/pkg/errors"

	"github.com/google/uuid"
)

func parseUUIDArg(raw interface{}, name string) (uuid.UUID, error) {
	switch v := raw.(type) {
	case string:
		if v == "" {
			return uuid.Nil, errors.Wrapf(errors.ErrInternal, "%s is required", name)
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, errors.Wrapf(err, "%s is invalid", name)
		}
		return id, nil
	case uuid.UUID:
		if v == uuid.Nil {
			return uuid.Nil, errors.Wrapf(errors.ErrInternal, "%s is required", name)
		}
		return v, nil
	default:
		return uuid.Nil, errors.Wrapf(errors.ErrInternal, "%s is required", name)
	}
}

// toUUID converts various types to UUID
func toUUID(raw interface{}) (uuid.UUID, error) {
	return parseUUIDArg(raw, "id")
}
