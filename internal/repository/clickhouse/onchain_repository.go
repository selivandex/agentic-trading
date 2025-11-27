package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"prometheus/internal/domain/onchain"
	"prometheus/pkg/errors"
)

// OnChainRepository implements onchain.Repository for ClickHouse
type OnChainRepository struct {
	conn driver.Conn
}

// NewOnChainRepository creates a new on-chain data repository
func NewOnChainRepository(conn driver.Conn) *OnChainRepository {
	return &OnChainRepository{conn: conn}
}

// ============================================================================
// Whale Movement Operations
// ============================================================================

// InsertWhaleMovement inserts a single whale movement
func (r *OnChainRepository) InsertWhaleMovement(ctx context.Context, movement *onchain.WhaleMovement) error {
	query := `
		INSERT INTO whale_movements (
			tx_hash, blockchain, from_address, to_address, token,
			amount, amount_usd, timestamp, from_label, to_label
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := r.conn.Exec(ctx, query,
		movement.TxHash,
		movement.Blockchain,
		movement.FromAddress,
		movement.ToAddress,
		movement.Token,
		movement.Amount,
		movement.AmountUSD,
		movement.Timestamp,
		movement.FromLabel,
		movement.ToLabel,
	)

	if err != nil {
		return errors.Wrap(err, "insert whale movement")
	}

	return nil
}

// InsertWhaleMovementBatch inserts multiple whale movements efficiently
func (r *OnChainRepository) InsertWhaleMovementBatch(ctx context.Context, movements []onchain.WhaleMovement) error {
	if len(movements) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO whale_movements (
			tx_hash, blockchain, from_address, to_address, token,
			amount, amount_usd, timestamp, from_label, to_label
		)
	`)
	if err != nil {
		return errors.Wrap(err, "prepare whale movements batch")
	}

	for _, m := range movements {
		err := batch.Append(
			m.TxHash, m.Blockchain, m.FromAddress, m.ToAddress, m.Token,
			m.Amount, m.AmountUSD, m.Timestamp, m.FromLabel, m.ToLabel,
		)
		if err != nil {
			return errors.Wrap(err, "append to whale movements batch")
		}
	}

	if err := batch.Send(); err != nil {
		return errors.Wrap(err, "send whale movements batch")
	}

	return nil
}

// GetRecentWhaleMovements retrieves recent large movements
func (r *OnChainRepository) GetRecentWhaleMovements(ctx context.Context, minAmount float64, limit int) ([]onchain.WhaleMovement, error) {
	query := `
		SELECT 
			tx_hash, blockchain, from_address, to_address, token,
			amount, amount_usd, timestamp, from_label, to_label
		FROM whale_movements
		WHERE amount_usd >= ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.conn.Query(ctx, query, minAmount, limit)
	if err != nil {
		return nil, errors.Wrap(err, "query recent whale movements")
	}
	defer rows.Close()

	var movements []onchain.WhaleMovement
	for rows.Next() {
		var m onchain.WhaleMovement
		if err := rows.Scan(
			&m.TxHash, &m.Blockchain, &m.FromAddress, &m.ToAddress, &m.Token,
			&m.Amount, &m.AmountUSD, &m.Timestamp, &m.FromLabel, &m.ToLabel,
		); err != nil {
			return nil, errors.Wrap(err, "scan whale movement")
		}
		movements = append(movements, m)
	}

	return movements, nil
}

// GetWhaleMovementsByBlockchain retrieves movements for a specific blockchain
func (r *OnChainRepository) GetWhaleMovementsByBlockchain(ctx context.Context, blockchain string, since time.Time) ([]onchain.WhaleMovement, error) {
	query := `
		SELECT 
			tx_hash, blockchain, from_address, to_address, token,
			amount, amount_usd, timestamp, from_label, to_label
		FROM whale_movements
		WHERE blockchain = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, blockchain, since)
	if err != nil {
		return nil, errors.Wrap(err, "query whale movements by blockchain")
	}
	defer rows.Close()

	var movements []onchain.WhaleMovement
	for rows.Next() {
		var m onchain.WhaleMovement
		if err := rows.Scan(
			&m.TxHash, &m.Blockchain, &m.FromAddress, &m.ToAddress, &m.Token,
			&m.Amount, &m.AmountUSD, &m.Timestamp, &m.FromLabel, &m.ToLabel,
		); err != nil {
			return nil, errors.Wrap(err, "scan whale movement")
		}
		movements = append(movements, m)
	}

	return movements, nil
}

// GetWhaleMovementsByToken retrieves large movements for a specific token
func (r *OnChainRepository) GetWhaleMovementsByToken(ctx context.Context, token string, minAmount float64, limit int) ([]onchain.WhaleMovement, error) {
	query := `
		SELECT 
			tx_hash, blockchain, from_address, to_address, token,
			amount, amount_usd, timestamp, from_label, to_label
		FROM whale_movements
		WHERE token = ? AND amount_usd >= ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.conn.Query(ctx, query, token, minAmount, limit)
	if err != nil {
		return nil, errors.Wrap(err, "query whale movements by token")
	}
	defer rows.Close()

	var movements []onchain.WhaleMovement
	for rows.Next() {
		var m onchain.WhaleMovement
		if err := rows.Scan(
			&m.TxHash, &m.Blockchain, &m.FromAddress, &m.ToAddress, &m.Token,
			&m.Amount, &m.AmountUSD, &m.Timestamp, &m.FromLabel, &m.ToLabel,
		); err != nil {
			return nil, errors.Wrap(err, "scan whale movement")
		}
		movements = append(movements, m)
	}

	return movements, nil
}

// ============================================================================
// Exchange Flow Operations
// ============================================================================

// InsertExchangeFlow inserts a single exchange flow record
func (r *OnChainRepository) InsertExchangeFlow(ctx context.Context, flow *onchain.ExchangeFlow) error {
	query := `
		INSERT INTO exchange_flows (
			exchange, blockchain, token, inflow_usd, outflow_usd, net_flow_usd, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	err := r.conn.Exec(ctx, query,
		flow.Exchange,
		flow.Blockchain,
		flow.Token,
		flow.InflowUSD,
		flow.OutflowUSD,
		flow.NetFlowUSD,
		flow.Timestamp,
	)

	if err != nil {
		return errors.Wrap(err, "insert exchange flow")
	}

	return nil
}

// InsertExchangeFlowBatch inserts multiple exchange flow records
func (r *OnChainRepository) InsertExchangeFlowBatch(ctx context.Context, flows []onchain.ExchangeFlow) error {
	if len(flows) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO exchange_flows (
			exchange, blockchain, token, inflow_usd, outflow_usd, net_flow_usd, timestamp
		)
	`)
	if err != nil {
		return errors.Wrap(err, "prepare exchange flows batch")
	}

	for _, f := range flows {
		err := batch.Append(
			f.Exchange, f.Blockchain, f.Token,
			f.InflowUSD, f.OutflowUSD, f.NetFlowUSD, f.Timestamp,
		)
		if err != nil {
			return errors.Wrap(err, "append to exchange flows batch")
		}
	}

	if err := batch.Send(); err != nil {
		return errors.Wrap(err, "send exchange flows batch")
	}

	return nil
}

// GetExchangeFlows retrieves flows for a specific exchange
func (r *OnChainRepository) GetExchangeFlows(ctx context.Context, exchange string, since time.Time) ([]onchain.ExchangeFlow, error) {
	query := `
		SELECT exchange, blockchain, token, inflow_usd, outflow_usd, net_flow_usd, timestamp
		FROM exchange_flows
		WHERE exchange = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, exchange, since)
	if err != nil {
		return nil, errors.Wrap(err, "query exchange flows")
	}
	defer rows.Close()

	var flows []onchain.ExchangeFlow
	for rows.Next() {
		var f onchain.ExchangeFlow
		if err := rows.Scan(
			&f.Exchange, &f.Blockchain, &f.Token,
			&f.InflowUSD, &f.OutflowUSD, &f.NetFlowUSD, &f.Timestamp,
		); err != nil {
			return nil, errors.Wrap(err, "scan exchange flow")
		}
		flows = append(flows, f)
	}

	return flows, nil
}

// GetExchangeFlowsByToken retrieves flows for a specific token across all exchanges
func (r *OnChainRepository) GetExchangeFlowsByToken(ctx context.Context, token string, since time.Time) ([]onchain.ExchangeFlow, error) {
	query := `
		SELECT exchange, blockchain, token, inflow_usd, outflow_usd, net_flow_usd, timestamp
		FROM exchange_flows
		WHERE token = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, token, since)
	if err != nil {
		return nil, errors.Wrap(err, "query exchange flows by token")
	}
	defer rows.Close()

	var flows []onchain.ExchangeFlow
	for rows.Next() {
		var f onchain.ExchangeFlow
		if err := rows.Scan(
			&f.Exchange, &f.Blockchain, &f.Token,
			&f.InflowUSD, &f.OutflowUSD, &f.NetFlowUSD, &f.Timestamp,
		); err != nil {
			return nil, errors.Wrap(err, "scan exchange flow")
		}
		flows = append(flows, f)
	}

	return flows, nil
}

// GetNetFlowSummary retrieves aggregated net flows across all exchanges
func (r *OnChainRepository) GetNetFlowSummary(ctx context.Context, since time.Time) ([]onchain.ExchangeFlow, error) {
	query := `
		SELECT 
			exchange,
			blockchain,
			token,
			sum(inflow_usd) as inflow_usd,
			sum(outflow_usd) as outflow_usd,
			sum(net_flow_usd) as net_flow_usd,
			max(timestamp) as timestamp
		FROM exchange_flows
		WHERE timestamp >= ?
		GROUP BY exchange, blockchain, token
		ORDER BY abs(net_flow_usd) DESC
		LIMIT 100
	`

	rows, err := r.conn.Query(ctx, query, since)
	if err != nil {
		return nil, errors.Wrap(err, "query net flow summary")
	}
	defer rows.Close()

	var flows []onchain.ExchangeFlow
	for rows.Next() {
		var f onchain.ExchangeFlow
		if err := rows.Scan(
			&f.Exchange, &f.Blockchain, &f.Token,
			&f.InflowUSD, &f.OutflowUSD, &f.NetFlowUSD, &f.Timestamp,
		); err != nil {
			return nil, errors.Wrap(err, "scan net flow summary")
		}
		flows = append(flows, f)
	}

	return flows, nil
}

// ============================================================================
// Network Metrics Operations
// ============================================================================

// InsertNetworkMetrics inserts network metrics for a blockchain
func (r *OnChainRepository) InsertNetworkMetrics(ctx context.Context, metrics *onchain.NetworkMetrics) error {
	query := `
		INSERT INTO network_metrics (
			blockchain, timestamp, active_addresses, transaction_count, avg_fee_usd,
			hash_rate, mempool_size, mempool_bytes,
			gas_price, gas_used, block_utilization
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := r.conn.Exec(ctx, query,
		metrics.Blockchain,
		metrics.Timestamp,
		metrics.ActiveAddresses,
		metrics.TransactionCount,
		metrics.AvgFeeUSD,
		metrics.HashRate,
		metrics.MempoolSize,
		metrics.MempoolBytes,
		metrics.GasPrice,
		metrics.GasUsed,
		metrics.BlockUtilization,
	)

	if err != nil {
		return errors.Wrap(err, "insert network metrics")
	}

	return nil
}

// GetLatestNetworkMetrics retrieves the most recent metrics for a blockchain
func (r *OnChainRepository) GetLatestNetworkMetrics(ctx context.Context, blockchain string) (*onchain.NetworkMetrics, error) {
	query := `
		SELECT 
			blockchain, timestamp, active_addresses, transaction_count, avg_fee_usd,
			hash_rate, mempool_size, mempool_bytes,
			gas_price, gas_used, block_utilization
		FROM network_metrics
		WHERE blockchain = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var m onchain.NetworkMetrics
	err := r.conn.QueryRow(ctx, query, blockchain).Scan(
		&m.Blockchain, &m.Timestamp, &m.ActiveAddresses, &m.TransactionCount, &m.AvgFeeUSD,
		&m.HashRate, &m.MempoolSize, &m.MempoolBytes,
		&m.GasPrice, &m.GasUsed, &m.BlockUtilization,
	)

	if err != nil {
		return nil, errors.Wrap(err, "query latest network metrics")
	}

	return &m, nil
}

// GetNetworkMetricsHistory retrieves historical metrics for a blockchain
func (r *OnChainRepository) GetNetworkMetricsHistory(ctx context.Context, blockchain string, since time.Time) ([]onchain.NetworkMetrics, error) {
	query := `
		SELECT 
			blockchain, timestamp, active_addresses, transaction_count, avg_fee_usd,
			hash_rate, mempool_size, mempool_bytes,
			gas_price, gas_used, block_utilization
		FROM network_metrics
		WHERE blockchain = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, blockchain, since)
	if err != nil {
		return nil, errors.Wrap(err, "query network metrics history")
	}
	defer rows.Close()

	var metrics []onchain.NetworkMetrics
	for rows.Next() {
		var m onchain.NetworkMetrics
		if err := rows.Scan(
			&m.Blockchain, &m.Timestamp, &m.ActiveAddresses, &m.TransactionCount, &m.AvgFeeUSD,
			&m.HashRate, &m.MempoolSize, &m.MempoolBytes,
			&m.GasPrice, &m.GasUsed, &m.BlockUtilization,
		); err != nil {
			return nil, errors.Wrap(err, "scan network metrics")
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// ============================================================================
// Miner Metrics Operations
// ============================================================================

// InsertMinerMetrics inserts miner metrics for a pool
func (r *OnChainRepository) InsertMinerMetrics(ctx context.Context, metrics *onchain.MinerMetrics) error {
	query := `
		INSERT INTO miner_metrics (
			pool_name, timestamp, hash_rate, hash_rate_share,
			btc_balance, btc_sent, btc_received, mpi
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := r.conn.Exec(ctx, query,
		metrics.PoolName,
		metrics.Timestamp,
		metrics.HashRate,
		metrics.HashRateShare,
		metrics.BTCBalance,
		metrics.BTCSent,
		metrics.BTCReceived,
		metrics.MPI,
	)

	if err != nil {
		return errors.Wrap(err, "insert miner metrics")
	}

	return nil
}

// InsertMinerMetricsBatch inserts multiple miner metrics records
func (r *OnChainRepository) InsertMinerMetricsBatch(ctx context.Context, metrics []onchain.MinerMetrics) error {
	if len(metrics) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO miner_metrics (
			pool_name, timestamp, hash_rate, hash_rate_share,
			btc_balance, btc_sent, btc_received, mpi
		)
	`)
	if err != nil {
		return errors.Wrap(err, "prepare miner metrics batch")
	}

	for _, m := range metrics {
		err := batch.Append(
			m.PoolName, m.Timestamp, m.HashRate, m.HashRateShare,
			m.BTCBalance, m.BTCSent, m.BTCReceived, m.MPI,
		)
		if err != nil {
			return errors.Wrap(err, "append to miner metrics batch")
		}
	}

	if err := batch.Send(); err != nil {
		return errors.Wrap(err, "send miner metrics batch")
	}

	return nil
}

// GetMinerActivity retrieves recent miner activity across all pools
func (r *OnChainRepository) GetMinerActivity(ctx context.Context, since time.Time) ([]onchain.MinerMetrics, error) {
	query := `
		SELECT pool_name, timestamp, hash_rate, hash_rate_share,
			   btc_balance, btc_sent, btc_received, mpi
		FROM miner_metrics
		WHERE timestamp >= ?
		ORDER BY timestamp DESC, hash_rate DESC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, since)
	if err != nil {
		return nil, errors.Wrap(err, "query miner activity")
	}
	defer rows.Close()

	var metrics []onchain.MinerMetrics
	for rows.Next() {
		var m onchain.MinerMetrics
		if err := rows.Scan(
			&m.PoolName, &m.Timestamp, &m.HashRate, &m.HashRateShare,
			&m.BTCBalance, &m.BTCSent, &m.BTCReceived, &m.MPI,
		); err != nil {
			return nil, errors.Wrap(err, "scan miner metrics")
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// GetMinerMetricsByPool retrieves metrics for a specific mining pool
func (r *OnChainRepository) GetMinerMetricsByPool(ctx context.Context, poolName string, since time.Time) ([]onchain.MinerMetrics, error) {
	query := `
		SELECT pool_name, timestamp, hash_rate, hash_rate_share,
			   btc_balance, btc_sent, btc_received, mpi
		FROM miner_metrics
		WHERE pool_name = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, poolName, since)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("query miner metrics for pool %s", poolName))
	}
	defer rows.Close()

	var metrics []onchain.MinerMetrics
	for rows.Next() {
		var m onchain.MinerMetrics
		if err := rows.Scan(
			&m.PoolName, &m.Timestamp, &m.HashRate, &m.HashRateShare,
			&m.BTCBalance, &m.BTCSent, &m.BTCReceived, &m.MPI,
		); err != nil {
			return nil, errors.Wrap(err, "scan miner metrics")
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// GetLatestMinerMetrics retrieves the most recent metrics for all pools
func (r *OnChainRepository) GetLatestMinerMetrics(ctx context.Context) ([]onchain.MinerMetrics, error) {
	query := `
		SELECT pool_name, timestamp, hash_rate, hash_rate_share,
			   btc_balance, btc_sent, btc_received, mpi
		FROM (
			SELECT *,
				   ROW_NUMBER() OVER (PARTITION BY pool_name ORDER BY timestamp DESC) as rn
			FROM miner_metrics
			WHERE timestamp >= now() - INTERVAL 24 HOUR
		)
		WHERE rn = 1
		ORDER BY hash_rate DESC
	`

	rows, err := r.conn.Query(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "query latest miner metrics")
	}
	defer rows.Close()

	var metrics []onchain.MinerMetrics
	for rows.Next() {
		var m onchain.MinerMetrics
		if err := rows.Scan(
			&m.PoolName, &m.Timestamp, &m.HashRate, &m.HashRateShare,
			&m.BTCBalance, &m.BTCSent, &m.BTCReceived, &m.MPI,
		); err != nil {
			return nil, errors.Wrap(err, "scan miner metrics")
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}


