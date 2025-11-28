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

// StoreFeatures stores extracted regime features for ML training/inference
func (r *RegimeRepository) StoreFeatures(ctx context.Context, features *regime.Features) error {
	query := `
		INSERT INTO regime_features (
			symbol, timestamp,
			atr_14, atr_pct, bb_width, historical_vol,
			adx, ema_9, ema_21, ema_55, ema_200, ema_alignment, higher_highs_count, lower_lows_count,
			volume_24h, volume_change_pct, volume_price_divergence,
			support_breaks, resistance_breaks, consolidation_periods,
			btc_dominance, correlation_tightness,
			funding_rate, funding_rate_ma, liquidations_24h,
			regime_label
		) VALUES (
			?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?,
			?
		)
	`

	err := r.conn.Exec(ctx, query,
		features.Symbol, features.Timestamp,
		// Volatility
		features.ATR14, features.ATRPct, features.BBWidth, features.HistoricalVol,
		// Trend
		features.ADX, features.EMA9, features.EMA21, features.EMA55, features.EMA200,
		features.EMAAlignment, features.HigherHighsCount, features.LowerLowsCount,
		// Volume
		features.Volume24h, features.VolumeChangePct, features.VolumePriceDivergence,
		// Structure
		features.SupportBreaks, features.ResistanceBreaks, features.ConsolidationPeriods,
		// Cross-asset
		features.BTCDominance, features.CorrelationTightness,
		// Derivatives
		features.FundingRate, features.FundingRateMA, features.Liquidations24h,
		// Label
		features.RegimeLabel,
	)

	if err != nil {
		return errors.Wrap(err, "failed to store regime features")
	}

	return nil
}

// GetLatestFeatures retrieves the latest extracted features for a symbol
func (r *RegimeRepository) GetLatestFeatures(ctx context.Context, symbol string) (*regime.Features, error) {
	query := `
		SELECT 
			symbol, timestamp,
			atr_14, atr_pct, bb_width, historical_vol,
			adx, ema_9, ema_21, ema_55, ema_200, ema_alignment, higher_highs_count, lower_lows_count,
			volume_24h, volume_change_pct, volume_price_divergence,
			support_breaks, resistance_breaks, consolidation_periods,
			btc_dominance, correlation_tightness,
			funding_rate, funding_rate_ma, liquidations_24h,
			regime_label
		FROM regime_features
		WHERE symbol = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	row := r.conn.QueryRow(ctx, query, symbol)

	var features regime.Features

	err := row.Scan(
		&features.Symbol, &features.Timestamp,
		// Volatility
		&features.ATR14, &features.ATRPct, &features.BBWidth, &features.HistoricalVol,
		// Trend
		&features.ADX, &features.EMA9, &features.EMA21, &features.EMA55, &features.EMA200,
		&features.EMAAlignment, &features.HigherHighsCount, &features.LowerLowsCount,
		// Volume
		&features.Volume24h, &features.VolumeChangePct, &features.VolumePriceDivergence,
		// Structure
		&features.SupportBreaks, &features.ResistanceBreaks, &features.ConsolidationPeriods,
		// Cross-asset
		&features.BTCDominance, &features.CorrelationTightness,
		// Derivatives
		&features.FundingRate, &features.FundingRateMA, &features.Liquidations24h,
		// Label
		&features.RegimeLabel,
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest features")
	}

	return &features, nil
}

// GetFeaturesHistory retrieves historical features for a symbol
func (r *RegimeRepository) GetFeaturesHistory(ctx context.Context, symbol string, since time.Time, limit int) ([]regime.Features, error) {
	query := `
		SELECT 
			symbol, timestamp,
			atr_14, atr_pct, bb_width, historical_vol,
			adx, ema_9, ema_21, ema_55, ema_200, ema_alignment, higher_highs_count, lower_lows_count,
			volume_24h, volume_change_pct, volume_price_divergence,
			support_breaks, resistance_breaks, consolidation_periods,
			btc_dominance, correlation_tightness,
			funding_rate, funding_rate_ma, liquidations_24h,
			regime_label
		FROM regime_features
		WHERE symbol = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.conn.Query(ctx, query, symbol, since, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get features history")
	}
	defer rows.Close()

	var featuresList []regime.Features

	for rows.Next() {
		var features regime.Features

		err := rows.Scan(
			&features.Symbol, &features.Timestamp,
			// Volatility
			&features.ATR14, &features.ATRPct, &features.BBWidth, &features.HistoricalVol,
			// Trend
			&features.ADX, &features.EMA9, &features.EMA21, &features.EMA55, &features.EMA200,
			&features.EMAAlignment, &features.HigherHighsCount, &features.LowerLowsCount,
			// Volume
			&features.Volume24h, &features.VolumeChangePct, &features.VolumePriceDivergence,
			// Structure
			&features.SupportBreaks, &features.ResistanceBreaks, &features.ConsolidationPeriods,
			// Cross-asset
			&features.BTCDominance, &features.CorrelationTightness,
			// Derivatives
			&features.FundingRate, &features.FundingRateMA, &features.Liquidations24h,
			// Label
			&features.RegimeLabel,
		)

		if err != nil {
			return nil, errors.Wrap(err, "failed to scan features row")
		}

		featuresList = append(featuresList, features)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate features rows")
	}

	return featuresList, nil
}
