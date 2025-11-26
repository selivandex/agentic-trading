package trading

import (
	"github.com/google/uuid"
	"prometheus/pkg/errors"
)

func parseUUIDArg(raw interface{}, name string) (uuid.UUID, error) {
	switch v := raw.(type) {
	case string:
		if v == "" {
			return uuid.Nil, errors.Wrapf(errors.ErrInternal, "%s is required", name)
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, errors.Wrapf(errors.ErrInternal, "%s is invalid: %w", name, err)
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
