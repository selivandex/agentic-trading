package websocket

import (
	"fmt"

	"github.com/google/uuid"

	"prometheus/internal/adapters/websocket/binance"
	"prometheus/internal/domain/exchange_account"
	"prometheus/pkg/logger"
)

// UserDataStreamFactory creates exchange-specific User Data WebSocket clients
type UserDataStreamFactory struct {
	logger *logger.Logger
}

// NewUserDataStreamFactory creates a new factory instance
func NewUserDataStreamFactory(log *logger.Logger) *UserDataStreamFactory {
	return &UserDataStreamFactory{
		logger: log,
	}
}

// Create instantiates an exchange-specific UserDataStreamer
func (f *UserDataStreamFactory) Create(
	exchange exchange_account.ExchangeType,
	accountID uuid.UUID,
	userID uuid.UUID,
	handler UserDataEventHandler,
	useTestnet bool,
) (UserDataStreamer, error) {
	switch exchange {
	case exchange_account.ExchangeBinance:
		return binance.NewUserDataClient(
			accountID,
			userID,
			string(exchange),
			handler,
			useTestnet,
			f.logger,
		), nil

	case exchange_account.ExchangeBybit:
		// TODO: Implement Bybit User Data client
		return nil, fmt.Errorf("bybit user data websocket not yet implemented")

	case exchange_account.ExchangeOKX:
		// TODO: Implement OKX User Data client
		return nil, fmt.Errorf("okx user data websocket not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported exchange for user data websocket: %s", exchange)
	}
}

