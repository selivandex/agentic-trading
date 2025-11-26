package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"prometheus/internal/domain/derivatives"
	"prometheus/pkg/errors"
)

// DerivativesRepository implements derivatives.Repository for ClickHouse
type DerivativesRepository struct {
	conn driver.Conn
}

// NewDerivativesRepository creates a new derivatives data repository
func NewDerivativesRepository(conn driver.Conn) *DerivativesRepository {
	return &DerivativesRepository{conn: conn}
}

// ============================================================================
// Options Snapshot Operations
// ============================================================================

// InsertOptionsSnapshot inserts an options market snapshot
func (r *DerivativesRepository) InsertOptionsSnapshot(ctx context.Context, snapshot *derivatives.OptionsSnapshot) error {
	query := `
		INSERT INTO options_snapshots (
			symbol, timestamp, call_oi, put_oi, total_oi, put_call_ratio,
			max_pain_price, max_pain_delta, gamma_exposure, gamma_flip,
			iv_index, iv_25d_call, iv_25d_put, iv_skew,
			large_call_vol, large_put_vol
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := r.conn.Exec(ctx, query,
		snapshot.Symbol,
		snapshot.Timestamp,
		snapshot.CallOI,
		snapshot.PutOI,
		snapshot.TotalOI,
		snapshot.PutCallRatio,
		snapshot.MaxPainPrice,
		snapshot.MaxPainDelta,
		snapshot.GammaExposure,
		snapshot.GammaFlip,
		snapshot.IVIndex,
		snapshot.IV25dCall,
		snapshot.IV25dPut,
		snapshot.IVSkew,
		snapshot.LargeCallVol,
		snapshot.LargePutVol,
	)

	if err != nil {
		return errors.Wrap(err, "insert options snapshot")
	}

	return nil
}

// GetLatestOptionsSnapshot retrieves the most recent options snapshot for a symbol
func (r *DerivativesRepository) GetLatestOptionsSnapshot(ctx context.Context, symbol string) (*derivatives.OptionsSnapshot, error) {
	query := `
		SELECT 
			symbol, timestamp, call_oi, put_oi, total_oi, put_call_ratio,
			max_pain_price, max_pain_delta, gamma_exposure, gamma_flip,
			iv_index, iv_25d_call, iv_25d_put, iv_skew,
			large_call_vol, large_put_vol
		FROM options_snapshots
		WHERE symbol = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var s derivatives.OptionsSnapshot
	err := r.conn.QueryRow(ctx, query, symbol).Scan(
		&s.Symbol, &s.Timestamp, &s.CallOI, &s.PutOI, &s.TotalOI, &s.PutCallRatio,
		&s.MaxPainPrice, &s.MaxPainDelta, &s.GammaExposure, &s.GammaFlip,
		&s.IVIndex, &s.IV25dCall, &s.IV25dPut, &s.IVSkew,
		&s.LargeCallVol, &s.LargePutVol,
	)

	if err != nil {
		return nil, errors.Wrap(err, "query latest options snapshot")
	}

	return &s, nil
}

// GetOptionsHistory retrieves historical options snapshots
func (r *DerivativesRepository) GetOptionsHistory(ctx context.Context, symbol string, since time.Time) ([]derivatives.OptionsSnapshot, error) {
	query := `
		SELECT 
			symbol, timestamp, call_oi, put_oi, total_oi, put_call_ratio,
			max_pain_price, max_pain_delta, gamma_exposure, gamma_flip,
			iv_index, iv_25d_call, iv_25d_put, iv_skew,
			large_call_vol, large_put_vol
		FROM options_snapshots
		WHERE symbol = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, symbol, since)
	if err != nil {
		return nil, errors.Wrap(err, "query options history")
	}
	defer rows.Close()

	var snapshots []derivatives.OptionsSnapshot
	for rows.Next() {
		var s derivatives.OptionsSnapshot
		if err := rows.Scan(
			&s.Symbol, &s.Timestamp, &s.CallOI, &s.PutOI, &s.TotalOI, &s.PutCallRatio,
			&s.MaxPainPrice, &s.MaxPainDelta, &s.GammaExposure, &s.GammaFlip,
			&s.IVIndex, &s.IV25dCall, &s.IV25dPut, &s.IVSkew,
			&s.LargeCallVol, &s.LargePutVol,
		); err != nil {
			return nil, errors.Wrap(err, "scan options snapshot")
		}
		snapshots = append(snapshots, s)
	}

	return snapshots, nil
}

// ============================================================================
// Options Flow Operations
// ============================================================================

// InsertOptionsFlow inserts a large options trade
func (r *DerivativesRepository) InsertOptionsFlow(ctx context.Context, flow *derivatives.OptionsFlow) error {
	query := `
		INSERT INTO options_flows (
			id, timestamp, symbol, side, strike, expiry, premium, size, spot, trade_type, sentiment
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := r.conn.Exec(ctx, query,
		flow.ID,
		flow.Timestamp,
		flow.Symbol,
		flow.Side,
		flow.Strike,
		flow.Expiry,
		flow.Premium,
		flow.Size,
		flow.Spot,
		flow.TradeType,
		flow.Sentiment,
	)

	if err != nil {
		return errors.Wrap(err, "insert options flow")
	}

	return nil
}

// GetLargeOptionsFlows retrieves large options trades above a premium threshold
func (r *DerivativesRepository) GetLargeOptionsFlows(ctx context.Context, symbol string, minPremium float64, limit int) ([]derivatives.OptionsFlow, error) {
	query := `
		SELECT id, timestamp, symbol, side, strike, expiry, premium, size, spot, trade_type, sentiment
		FROM options_flows
		WHERE symbol = ? AND premium >= ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.conn.Query(ctx, query, symbol, minPremium, limit)
	if err != nil {
		return nil, errors.Wrap(err, "query large options flows")
	}
	defer rows.Close()

	var flows []derivatives.OptionsFlow
	for rows.Next() {
		var f derivatives.OptionsFlow
		if err := rows.Scan(
			&f.ID, &f.Timestamp, &f.Symbol, &f.Side, &f.Strike, &f.Expiry,
			&f.Premium, &f.Size, &f.Spot, &f.TradeType, &f.Sentiment,
		); err != nil {
			return nil, errors.Wrap(err, "scan options flow")
		}
		flows = append(flows, f)
	}

	return flows, nil
}

// GetRecentOptionsFlows retrieves recent options flows since a timestamp
func (r *DerivativesRepository) GetRecentOptionsFlows(ctx context.Context, symbol string, since time.Time) ([]derivatives.OptionsFlow, error) {
	query := `
		SELECT id, timestamp, symbol, side, strike, expiry, premium, size, spot, trade_type, sentiment
		FROM options_flows
		WHERE symbol = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, symbol, since)
	if err != nil {
		return nil, errors.Wrap(err, "query recent options flows")
	}
	defer rows.Close()

	var flows []derivatives.OptionsFlow
	for rows.Next() {
		var f derivatives.OptionsFlow
		if err := rows.Scan(
			&f.ID, &f.Timestamp, &f.Symbol, &f.Side, &f.Strike, &f.Expiry,
			&f.Premium, &f.Size, &f.Spot, &f.TradeType, &f.Sentiment,
		); err != nil {
			return nil, errors.Wrap(err, "scan options flow")
		}
		flows = append(flows, f)
	}

	return flows, nil
}

