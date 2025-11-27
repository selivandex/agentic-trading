package onchain

import "time"

// WhaleMovement represents a large wallet transfer (typically >$1M)
// Used to track significant on-chain movements that may impact market
type WhaleMovement struct {
	TxHash      string    `ch:"tx_hash"`
	Blockchain  string    `ch:"blockchain"` // bitcoin, ethereum, bsc, etc.
	FromAddress string    `ch:"from_address"`
	ToAddress   string    `ch:"to_address"`
	Token       string    `ch:"token"`      // BTC, ETH, USDT, etc.
	Amount      float64   `ch:"amount"`     // Token amount
	AmountUSD   float64   `ch:"amount_usd"` // USD value at time of transfer
	Timestamp   time.Time `ch:"timestamp"`

	// Optional metadata
	FromLabel string `ch:"from_label"` // Exchange name, known wallet, etc.
	ToLabel   string `ch:"to_label"`
}

// ExchangeFlow represents net inflow/outflow to/from exchanges
// Aggregated data showing exchange reserve changes
type ExchangeFlow struct {
	Exchange   string    `ch:"exchange"`     // binance, coinbase, kraken
	Blockchain string    `ch:"blockchain"`   // bitcoin, ethereum
	Token      string    `ch:"token"`        // BTC, ETH, USDT
	InflowUSD  float64   `ch:"inflow_usd"`   // Money flowing into exchange
	OutflowUSD float64   `ch:"outflow_usd"`  // Money flowing out of exchange
	NetFlowUSD float64   `ch:"net_flow_usd"` // Net flow (inflow - outflow)
	Timestamp  time.Time `ch:"timestamp"`
}

// NetworkMetrics represents blockchain health and activity metrics
// Used to gauge network usage and potential congestion
type NetworkMetrics struct {
	Blockchain string    `ch:"blockchain"`
	Timestamp  time.Time `ch:"timestamp"`

	// Common metrics (all blockchains)
	ActiveAddresses  uint32  `ch:"active_addresses"`  // 24h active addresses
	TransactionCount uint32  `ch:"transaction_count"` // 24h transaction count
	AvgFeeUSD        float64 `ch:"avg_fee_usd"`       // Average transaction fee

	// Bitcoin-specific
	HashRate     float64 `ch:"hash_rate"`     // Network hash rate (TH/s)
	MempoolSize  uint32  `ch:"mempool_size"`  // Pending transactions
	MempoolBytes uint64  `ch:"mempool_bytes"` // Mempool size in bytes

	// Ethereum-specific
	GasPrice         float64 `ch:"gas_price"`         // Average gas price (gwei)
	GasUsed          uint64  `ch:"gas_used"`          // Gas used in 24h
	BlockUtilization float64 `ch:"block_utilization"` // % of gas limit used
}

// MinerMetrics represents mining pool behavior and holdings
// Used to track miner selling pressure and accumulation
type MinerMetrics struct {
	PoolName  string    `ch:"pool_name"` // Mining pool identifier
	Timestamp time.Time `ch:"timestamp"`

	// Mining power
	HashRate      float64 `ch:"hash_rate"`       // Pool hash rate (TH/s)
	HashRateShare float64 `ch:"hash_rate_share"` // % of network hash rate

	// Bitcoin holdings
	BTCBalance  float64 `ch:"btc_balance"`  // Current pool BTC balance
	BTCSent     float64 `ch:"btc_sent"`     // BTC sent (24h)
	BTCReceived float64 `ch:"btc_received"` // BTC received (24h, mining rewards)

	// Miner Position Index (MPI) - ratio of miner outflows to 1-year MA
	MPI float64 `ch:"mpi"` // >2.0 = high selling pressure, <0.5 = accumulation
}


