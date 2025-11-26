package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"prometheus/internal/domain/regime"
	"prometheus/pkg/errors"
)

// RegimeRepository implements regime.Repository for ClickHouse
type RegimeRepository struct {
	conn driver.Conn
}

// NewRegimeRepository creates a new regime repository
func NewRegimeRepository(conn driver.Conn) *RegimeRepository {
	return &RegimeRepository{conn: conn}
}

// Store stores a new market regime detection
func (r *RegimeRepository) Store(ctx context.Context, reg *regime.MarketRegime) error {
	query := `
		INSERT INTO market_regimes (
			symbol, timestamp, regime, confidence, volatility, trend,
			atr_14, adx, bb_width, volume_24h, volume_change
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?
		)
	`

	err := r.conn.Exec(ctx, query,
		reg.Symbol,
		reg.Timestamp,
		reg.Regime.String(),
		reg.Confidence,
		reg.Volatility.String(),
		reg.Trend.String(),
		reg.ATR14,
		reg.ADX,
		reg.BBWidth,
		reg.Volume24h,
		reg.VolumeChange,
	)

	if err != nil {
		return errors.Wrap(err, "failed to store market regime")
	}

	return nil
}

// GetLatest retrieves the latest regime for a symbol
func (r *RegimeRepository) GetLatest(ctx context.Context, symbol string) (*regime.MarketRegime, error) {
	query := `
		SELECT 
			symbol, timestamp, regime, confidence, volatility, trend,
			atr_14, adx, bb_width, volume_24h, volume_change
		FROM market_regimes
		WHERE symbol = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	row := r.conn.QueryRow(ctx, query, symbol)

	var reg regime.MarketRegime
	var regimeStr, volatilityStr, trendStr string

	err := row.Scan(
		&reg.Symbol,
		&reg.Timestamp,
		&regimeStr,
		&reg.Confidence,
		&volatilityStr,
		&trendStr,
		&reg.ATR14,
		&reg.ADX,
		&reg.BBWidth,
		&reg.Volume24h,
		&reg.VolumeChange,
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest regime")
	}

	reg.Regime = regime.RegimeType(regimeStr)
	reg.Volatility = regime.VolLevel(volatilityStr)
	reg.Trend = regime.TrendType(trendStr)

	return &reg, nil
}

// GetHistory retrieves regime history for a symbol since a given time
func (r *RegimeRepository) GetHistory(ctx context.Context, symbol string, since time.Time) ([]regime.MarketRegime, error) {
	query := `
		SELECT 
			symbol, timestamp, regime, confidence, volatility, trend,
			atr_14, adx, bb_width, volume_24h, volume_change
		FROM market_regimes
		WHERE symbol = ? AND timestamp >= ?
		ORDER BY timestamp DESC
	`

	rows, err := r.conn.Query(ctx, query, symbol, since)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get regime history")
	}
	defer rows.Close()

	var regimes []regime.MarketRegime

	for rows.Next() {
		var reg regime.MarketRegime
		var regimeStr, volatilityStr, trendStr string

		err := rows.Scan(
			&reg.Symbol,
			&reg.Timestamp,
			&regimeStr,
			&reg.Confidence,
			&volatilityStr,
			&trendStr,
			&reg.ATR14,
			&reg.ADX,
			&reg.BBWidth,
			&reg.Volume24h,
			&reg.VolumeChange,
		)

		if err != nil {
			return nil, errors.Wrap(err, "failed to scan regime row")
		}

		reg.Regime = regime.RegimeType(regimeStr)
		reg.Volatility = regime.VolLevel(volatilityStr)
		reg.Trend = regime.TrendType(trendStr)

		regimes = append(regimes, reg)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate regime rows")
	}

	return regimes, nil
}

