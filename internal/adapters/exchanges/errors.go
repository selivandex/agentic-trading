package exchanges

import "errors"

var (
	// ErrNotSupported is returned when the exchange does not support the requested feature.
	ErrNotSupported = errors.New("operation not supported by exchange")

	// ErrInvalidRequest indicates validation failures before hitting exchange API.
	ErrInvalidRequest = errors.New("invalid exchange request")

	// ErrRateLimited indicates HTTP 429 or throttling.
	ErrRateLimited = errors.New("exchange rate limited the request")
)

