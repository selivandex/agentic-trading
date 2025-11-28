<!-- @format -->

# Machine Learning in Prometheus Trading System

This document outlines ML opportunities, architectures, and implementation guidelines for the Prometheus autonomous trading system.

---

## Table of Contents

1. [Current State](#current-state)
2. [ML Opportunities](#ml-opportunities)
3. [Architecture](#architecture)
4. [Implementation Guidelines](#implementation-guidelines)
5. [Training Pipeline](#training-pipeline)
6. [Model Monitoring](#model-monitoring)
7. [Roadmap](#roadmap)

---

## Current State

### Implemented Components

#### 1. Regime Detector (Phase 4)

- **Location**: `internal/ml/regime/`
- **Model Type**: Random Forest / XGBoost → ONNX
- **Purpose**: Classify current market regime
- **Input Features** (22):
  - Volatility: ATR14, ATR%, Bollinger Bandwidth, Historical Vol
  - Trend: EMA slopes, ADX, Directional Movement
  - Volume: Volume trend, Volume/Price divergence
  - Structure: Support/resistance breaks, consolidation periods
  - Derivatives: Funding rates, liquidations
- **Output**:
  - Regime classification (trend_up, trend_down, range_bound, high_volatility, consolidation)
  - Confidence score
  - Probability distribution across all regimes
- **Training**: Offline training on historical data, manual labeling initially
- **Deployment**: ONNX model loaded at runtime
- **Integration**: Used by `regime_detector_ml` worker, feeds into agent decision-making

#### 2. ONNX Runtime Infrastructure

- **Location**: `internal/ml/onnx_runtime.go`
- **Purpose**: Universal ONNX model loader and inference engine
- **Features**:
  - Dynamic tensor creation
  - Multi-output support
  - Automatic cleanup
  - Thread-safe inference

---

## ML Opportunities

Below is a comprehensive list of ML applications in trading systems, ranked by priority and potential impact.

### 1. Anomaly Detection ⭐⭐⭐

**Purpose**: Detect unusual market behavior that may signal risk or opportunity

**Use Cases**:

- Flash crashes / pump & dump schemes
- Abnormal volume spikes
- Unusual correlation breakdowns
- Funding rate anomalies (derivatives market stress)
- Liquidity drying up (order book depth collapse)
- Whale activity (large orders/trades)

**Model Architecture**:

- **Algorithm**: Isolation Forest or Autoencoder
- **Input Features**:
  - Volume deviation (z-score from MA)
  - Price change velocity
  - Bid-ask spread widening
  - Funding rate deviation
  - Liquidation volume spike
  - Order book imbalance
- **Output**:
  - `is_anomaly` (boolean)
  - `anomaly_score` (0-1, higher = more anomalous)
  - `anomaly_type` (volume, price, liquidity, derivatives)

**Integration**:

```go
// Tool for agents
tool.Define("detect_market_anomalies",
    "Check if current market shows unusual behavior",
    func(ctx tool.Context, args struct{ Symbol string }) {
        result := anomalyDetector.Detect(features)
        if result.IsAnomaly && result.Score > 0.8 {
            return "⚠️ High anomaly detected, consider reducing exposure"
        }
    })
```

**Training**:

- Unsupervised learning on "normal" market conditions
- Label known anomalous events for validation
- Retrain weekly on recent data

**Priority**: **HIGH** — protects against extreme events, quick ROI

---

### 2. Volatility Forecasting ⭐⭐⭐

**Purpose**: Predict future volatility for dynamic position sizing and risk management

**Use Cases**:

- Adaptive position sizing (smaller size in high vol forecasts)
- Dynamic stop-loss placement (wider stops in high vol)
- Portfolio rebalancing trigger (reduce exposure before vol spike)
- Option pricing input (if trading options later)

**Model Architecture**:

- **Algorithm**: GARCH, EGARCH, or LSTM/GRU
- **Input**: Historical returns (1h, 4h, 1d) for past 30-90 days
- **Output**: Volatility forecast for next 6h, 12h, 24h, 48h
- **Metrics**: RMSE against realized volatility

**Integration**:

```go
// Risk engine uses forecast
func (e *Engine) CalculatePositionSize(account *domain.Account, signal *domain.Signal) float64 {
    volForecast := e.volForecaster.Predict(signal.Symbol, "24h")

    // Scale down if volatility spike expected
    baseSizeUSD := account.Capital * e.config.MaxPositionSizePercent
    volMultiplier := 1.0
    if volForecast.VolatilityPct > e.config.HighVolThreshold {
        volMultiplier = 0.6 // Reduce size by 40%
    }

    return baseSizeUSD * volMultiplier
}
```

**Training**:

- Supervised learning: past returns → realized future volatility
- Train on multiple symbols (BTC, ETH, major alts)
- Retrain monthly

**Priority**: **HIGH** — critical for risk management

---

### 3. Correlation Regime Detector ⭐⭐

**Purpose**: Detect when asset correlations change, affecting diversification benefits

**Use Cases**:

- When correlations spike (crisis mode) → reduce total exposure
- When correlations disperse → can hold more positions
- Identify uncorrelated pairs for better diversification
- Detect "risk-on / risk-off" regime shifts

**Model Architecture**:

- **Algorithm**: Clustering (K-means, HDBSCAN) + Change Point Detection
- **Input**: Rolling correlation matrix (30 assets, 7-day window)
- **Output**:
  - `correlation_regime` (tight, normal, dispersed)
  - `average_correlation` (0-1)
  - `regime_change_probability`

**Integration**:

```go
// Portfolio manager adjusts concentration
func (pm *PortfolioManager) GetMaxPositions() int {
    corrRegime := pm.corrAnalyzer.GetCurrentRegime()

    switch corrRegime.Regime {
    case "tight":
        return 3  // High correlation → fewer positions
    case "normal":
        return 5
    case "dispersed":
        return 8  // Low correlation → more positions
    }
}
```

**Training**:

- Unsupervised clustering on historical correlation matrices
- Label regimes based on market conditions (bull, bear, crisis)

**Priority**: **MEDIUM-HIGH** — improves portfolio diversification

---

### 4. Liquidity Predictor ⭐⭐

**Purpose**: Forecast upcoming low-liquidity periods to avoid slippage

**Use Cases**:

- Avoid large trades during predicted low-liquidity windows
- Adjust order sizing based on expected liquidity
- Warn agents before weekends/holidays
- Choose liquid vs illiquid assets

**Model Architecture**:

- **Algorithm**: Time series forecasting (Prophet, LSTM)
- **Input**:
  - Historical volume by hour-of-day, day-of-week
  - Holiday calendar
  - Recent volume trend
- **Output**: Expected volume for next 6h, 12h, 24h

**Integration**:

```go
// Execution engine checks liquidity before trading
func (e *Executor) ExecuteOrder(order *domain.Order) error {
    liquidityForecast := e.liquidityPredictor.Predict(order.Symbol, "6h")

    if liquidityForecast.VolumeUSD < order.SizeUSD * 10 {
        return fmt.Errorf("insufficient predicted liquidity: wait for better conditions")
    }

    return e.exchange.PlaceOrder(order)
}
```

**Training**:

- Historical volume data grouped by time features
- Separate models per symbol (BTC, ETH, alts)

**Priority**: **MEDIUM** — prevents slippage and stuck positions

---

### 5. Trade Quality Classifier ⭐⭐

**Purpose**: Learn from own trading history to filter low-quality setups

**Use Cases**:

- Predict probability of profitable trade before entry
- Filter out setups similar to past losses
- Continuous improvement loop
- Identify which features correlate with success

**Model Architecture**:

- **Algorithm**: XGBoost / LightGBM (binary classification)
- **Input Features**:
  - Entry conditions: RSI, ATR, regime, volume
  - Context: time of day, current drawdown, portfolio concentration
  - Signal source: which agent recommended it
- **Output**:
  - `win_probability` (0-1)
  - `expected_return` (estimated R-multiple)
  - Feature importance

**Integration**:

```go
// Risk engine filters low-quality trades
func (e *Engine) ApproveSignal(signal *domain.Signal) (bool, string) {
    quality := e.qualityClassifier.Predict(signal)

    if quality.WinProbability < 0.45 {
        return false, "Low win probability based on historical patterns"
    }

    return true, ""
}
```

**Training**:

- Supervised learning on own closed trades
- Label: profit/loss after 24h, 48h
- Retrain weekly as more trades accumulate

**Priority**: **MEDIUM** — requires trading history, improves over time

---

### 6. Pattern Recognition ⭐

**Purpose**: Automatically detect technical chart patterns

**Use Cases**:

- Head & Shoulders, Double Top/Bottom
- Triangles, Wedges, Channels
- Support/resistance levels
- Breakout confirmations

**Model Architecture**:

- **Algorithm**: CNN on candlestick images OR classical ML on features
- **Input**:
  - Option A: Candlestick chart as image (100x100 pixels)
  - Option B: Sequence of OHLC + volumes (50-100 bars)
- **Output**:
  - `pattern_type` (h&s, double_top, triangle_ascending, etc.)
  - `confidence` (0-1)
  - `breakout_direction` (up, down)

**Integration**:

```go
// Tool for Technical Analyst agent
tool.Define("detect_chart_patterns",
    "Scan for classical chart patterns",
    func(ctx tool.Context, args struct{ Symbol string }) {
        patterns := patternRecognizer.Detect(symbol)
        // Return detected patterns for agent reasoning
    })
```

**Training**:

- Synthetic data generation (draw patterns programmatically)
- Manual labeling of real historical patterns
- Data augmentation (slight variations)

**Priority**: **MEDIUM-LOW** — nice to have, not critical

---

### 7. Optimal Execution (RL) ⭐

**Purpose**: Learn optimal order splitting and timing to minimize slippage

**Use Cases**:

- Large orders → split into smaller chunks
- Choose best execution times (high liquidity windows)
- Multi-exchange routing (execute where slippage is lowest)
- Adaptive algorithms (TWAP, VWAP, aggressive vs passive)

**Model Architecture**:

- **Algorithm**: Reinforcement Learning (DDPG, TD3, PPO)
- **State**:
  - Order size remaining
  - Current order book (bids/asks depth)
  - Recent trade volume and direction
  - Time of day
- **Action**:
  - Order size to place now (0-100% of remaining)
  - Limit price (aggressive/passive)
  - Exchange selection
- **Reward**: -slippage - fees - market impact

**Integration**:

```go
// Execution engine uses RL policy
func (e *Executor) ExecuteLargeOrder(order *domain.Order) error {
    policy := e.rlExecutor.GetPolicy(order)

    for order.RemainingSize > 0 {
        action := policy.Decide(getCurrentState())
        chunkSize := action.Size

        e.exchange.PlaceOrder(chunkSize, action.LimitPrice)
        time.Sleep(action.WaitTime)
    }
}
```

**Training**:

- Simulated environment with historical order book data
- Offline RL training (batch)
- Deploy when Sharpe ratio > baseline (TWAP)

**Priority**: **MEDIUM-LOW** — complex, only needed for large orders

---

### 8. Sentiment Analysis ⭐

**Purpose**: Gauge market sentiment from news and social media

**Use Cases**:

- Contrarian signals (extreme fear → buy, extreme greed → sell)
- Event detection (regulatory news, hacks, major announcements)
- Filter trades: avoid longs in extreme negative sentiment
- Aggregate Fear & Greed Index from multiple sources

**Model Architecture**:

- **Algorithm**: Transformer (BERT, DistilBERT fine-tuned on financial text)
- **Input**:
  - News headlines (CoinDesk, CoinTelegraph, etc.)
  - Twitter/X posts (crypto influencers)
  - Reddit posts (r/cryptocurrency, r/bitcoin)
- **Output**:
  - `sentiment_score` (-1 to +1)
  - `sentiment_label` (very_bearish, bearish, neutral, bullish, very_bullish)
  - `event_detected` (boolean + event type)

**Integration**:

```go
// Tool for agents
tool.Define("get_market_sentiment",
    "Analyze current sentiment from news and social media",
    func(ctx tool.Context, args struct{ Symbol string }) {
        sentiment := sentimentAnalyzer.Analyze(symbol)
        if sentiment.Score < -0.7 {
            return "Extreme fear detected, potential buy opportunity (contrarian)"
        }
    })
```

**Training**:

- Fine-tune pre-trained BERT on financial text
- Label dataset: CryptoSentiment, StockTwits, etc.
- Continuous learning from feedback

**Priority**: **LOW-MEDIUM** — requires external data, less reliable than price-based signals

---

## Architecture

### Directory Structure

```
internal/ml/
├── onnx_runtime.go              # Universal ONNX loader (exists)
├── README.md
├── regime/                      # Regime detection (exists)
│   ├── classifier.go
│   └── classifier_test.go
├── anomaly/                     # NEW
│   ├── detector.go
│   └── detector_test.go
├── volatility/                  # NEW
│   ├── forecaster.go
│   └── forecaster_test.go
├── correlation/                 # NEW
│   ├── analyzer.go
│   └── analyzer_test.go
├── liquidity/                   # NEW
│   ├── predictor.go
│   └── predictor_test.go
├── quality/                     # NEW
│   ├── classifier.go
│   └── classifier_test.go
├── patterns/                    # NEW
│   ├── recognizer.go
│   └── recognizer_test.go
├── execution/                   # NEW (RL-based)
│   ├── optimizer.go
│   └── optimizer_test.go
└── sentiment/                   # NEW
    ├── analyzer.go
    └── analyzer_test.go

scripts/ml/
├── regime/                      # Training scripts (exist)
│   ├── train_model.py
│   ├── label_data.py
│   └── requirements.txt
├── anomaly/                     # NEW
│   ├── train_model.py
│   └── requirements.txt
├── volatility/                  # NEW
│   ├── train_garch.py
│   └── train_lstm.py
├── correlation/                 # NEW
│   └── train_model.py
├── liquidity/                   # NEW
│   └── train_model.py
├── quality/                     # NEW
│   └── train_model.py
├── patterns/                    # NEW
│   ├── generate_synthetic.py   # Data augmentation
│   └── train_model.py
├── execution/                   # NEW
│   ├── train_rl.py
│   └── backtest_env.py
└── sentiment/                   # NEW
    ├── fine_tune_bert.py
    └── requirements.txt

models/                          # Deployed ONNX models
├── regime_detector.onnx         # Exists
├── anomaly_detector.onnx        # NEW
├── volatility_forecaster.onnx   # NEW
├── correlation_analyzer.onnx    # NEW
├── liquidity_predictor.onnx     # NEW
├── trade_quality.onnx           # NEW
├── pattern_recognizer.onnx      # NEW
├── execution_policy.onnx        # NEW
└── sentiment_analyzer.onnx      # NEW
```

### Model Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│                    TRAINING PIPELINE                         │
│  (Offline, Python/scikit-learn/PyTorch)                     │
└──────────────┬──────────────────────────────────────────────┘
               │
               ▼
        ┌─────────────┐
        │ ClickHouse  │ ──► Feature extraction
        │  (Raw Data) │
        └─────────────┘
               │
               ▼
        ┌─────────────┐
        │   Train &   │ ──► sklearn, XGBoost, PyTorch
        │  Validate   │
        └─────────────┘
               │
               ▼
        ┌─────────────┐
        │   Export    │ ──► ONNX format
        │    ONNX     │
        └─────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────┐
│                  INFERENCE (Go Runtime)                       │
│                                                               │
│  ┌────────────┐       ┌────────────┐       ┌────────────┐  │
│  │  Worker    │──────▶│ ML Service │◀──────│  Tool      │  │
│  │  (cron)    │       │  (ONNX)    │       │  (agent)   │  │
│  └────────────┘       └────────────┘       └────────────┘  │
│                              │                                │
│                              ▼                                │
│                       ┌─────────────┐                        │
│                       │  Postgres   │                        │
│                       │  (Results)  │                        │
│                       └─────────────┘                        │
└──────────────────────────────────────────────────────────────┘
```

### Integration Patterns

#### Pattern 1: Worker-based (Scheduled Inference)

```go
// internal/workers/analysis/anomaly_detector.go
type AnomalyDetectorWorker struct {
    detector   *anomaly.Detector
    marketRepo market_data.Repository
    eventBus   *kafka.Producer
}

func (w *AnomalyDetectorWorker) Run(ctx context.Context) error {
    // Run every 5 minutes
    ticker := time.NewTicker(5 * time.Minute)

    for {
        select {
        case <-ticker.C:
            result := w.detector.Detect(features)
            if result.IsAnomaly {
                w.eventBus.Publish("market.anomaly_detected", result)
            }
        case <-ctx.Done():
            return nil
        }
    }
}
```

#### Pattern 2: Tool-based (On-demand Inference)

```go
// internal/tools/ml_tools.go
func NewMLTools(
    regimeClassifier *regime.Classifier,
    anomalyDetector *anomaly.Detector,
    volForecaster *volatility.Forecaster,
) []tool.Tool {
    return []tool.Tool{
        tool.Define("get_market_regime", "Classify current market regime",
            func(ctx tool.Context, args struct{ Symbol string }) {...}),

        tool.Define("check_anomalies", "Detect market anomalies",
            func(ctx tool.Context, args struct{ Symbol string }) {...}),

        tool.Define("forecast_volatility", "Predict future volatility",
            func(ctx tool.Context, args struct{
                Symbol  string
                Horizon string // "6h", "24h", "48h"
            }) {...}),
    }
}
```

#### Pattern 3: Risk Engine Integration

```go
// internal/risk/engine.go
type Engine struct {
    volForecaster     *volatility.Forecaster
    anomalyDetector   *anomaly.Detector
    qualityClassifier *quality.Classifier
}

func (e *Engine) ApproveSignal(signal *domain.Signal) (bool, string) {
    // Check 1: Anomaly detection
    anomaly := e.anomalyDetector.Detect(signal.Symbol)
    if anomaly.IsAnomaly && anomaly.Score > 0.8 {
        return false, "High market anomaly detected"
    }

    // Check 2: Volatility forecast
    vol := e.volForecaster.Predict(signal.Symbol, "24h")
    if vol.VolatilityPct > 10.0 {
        return false, "Extreme volatility forecasted"
    }

    // Check 3: Trade quality
    quality := e.qualityClassifier.Predict(signal)
    if quality.WinProbability < 0.45 {
        return false, "Low predicted win probability"
    }

    return true, ""
}
```

---

## Implementation Guidelines

### 1. Model Development Workflow

```bash
# Step 1: Collect training data
make clickhouse-query QUERY="SELECT * FROM market_features WHERE timestamp > now() - 30d"

# Step 2: Train model (Python)
cd scripts/ml/anomaly
python train_model.py --data ../../data/features.csv --output anomaly_detector.onnx

# Step 3: Validate model
python validate_model.py --model anomaly_detector.onnx --test-data test.csv

# Step 4: Deploy to models/ directory
cp anomaly_detector.onnx ../../../models/

# Step 5: Implement Go wrapper
# Edit internal/ml/anomaly/detector.go

# Step 6: Write tests
go test ./internal/ml/anomaly/...

# Step 7: Integrate into system
# Add to workers or tools
```

### 2. ONNX Export Standards

All models must be exported to ONNX format:

**Python (scikit-learn)**:

```python
from skl2onnx import convert_sklearn
from skl2onnx.common.data_types import FloatTensorType

# Train model
model = RandomForestClassifier()
model.fit(X, y)

# Export to ONNX
initial_type = [('float_input', FloatTensorType([None, X.shape[1]]))]
onnx_model = convert_sklearn(model, initial_types=initial_type)

with open("model.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
```

**Python (PyTorch)**:

```python
import torch

# Train model
model = MyNeuralNet()
model.train()

# Export to ONNX
dummy_input = torch.randn(1, input_size)
torch.onnx.export(
    model,
    dummy_input,
    "model.onnx",
    input_names=['input'],
    output_names=['output'],
    dynamic_axes={'input': {0: 'batch_size'}}
)
```

### 3. Go Integration Template

```go
package mymodel

import (
    "context"
    "fmt"
    "github.com/yalue/onnxruntime_go"
    "prometheus/internal/ml"
)

type Predictor struct {
    runtime *ml.ONNXRuntime
}

func NewPredictor(modelPath string) (*Predictor, error) {
    runtime, err := ml.NewONNXRuntime(modelPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load model: %w", err)
    }

    return &Predictor{runtime: runtime}, nil
}

type Input struct {
    Feature1 float64
    Feature2 float64
    // ... more features
}

type Output struct {
    Prediction float64
    Confidence float64
}

func (p *Predictor) Predict(ctx context.Context, input *Input) (*Output, error) {
    // Convert input to tensor
    inputTensor := []float32{
        float32(input.Feature1),
        float32(input.Feature2),
    }

    // Run inference
    outputs, err := p.runtime.Run([][]float32{inputTensor})
    if err != nil {
        return nil, fmt.Errorf("inference failed: %w", err)
    }

    return &Output{
        Prediction: float64(outputs[0][0]),
        Confidence: float64(outputs[1][0]),
    }, nil
}

func (p *Predictor) Close() error {
    return p.runtime.Close()
}
```

### 4. Feature Engineering Standards

All ML features should be:

- **Normalized**: Scale to [0, 1] or standardize (mean=0, std=1)
- **Stationary**: Use returns, not raw prices
- **Lookback-aware**: Don't use future data (no look-ahead bias)
- **Missing-value handled**: Impute or drop gracefully

Example feature calculation:

```go
func CalculateFeatures(candles []domain.OHLCV) *Features {
    return &Features{
        // Normalized volatility (0-1 scale)
        ATRPct: indicators.ATR(candles, 14) / candles[len(candles)-1].Close,

        // Z-score of volume (standardized)
        VolumeZScore: (currentVol - meanVol) / stdVol,

        // Stationary: returns, not prices
        Return1h: (candles[len(candles)-1].Close - candles[len(candles)-2].Close) / candles[len(candles)-2].Close,

        // Bounded: RSI already 0-100, normalize to 0-1
        RSI: indicators.RSI(candles, 14) / 100.0,
    }
}
```

### 5. Model Versioning

Use semantic versioning for models:

```
models/
├── anomaly_detector_v1.0.0.onnx
├── anomaly_detector_v1.1.0.onnx  # Minor update (retrained)
└── anomaly_detector_v2.0.0.onnx  # Major update (architecture change)
```

Config file tracks active versions:

```yaml
# config/ml_models.yaml
models:
  regime_detector:
    version: "1.0.0"
    path: "models/regime_detector_v1.0.0.onnx"

  anomaly_detector:
    version: "1.1.0"
    path: "models/anomaly_detector_v1.1.0.onnx"
```

### 6. Testing Standards

Every ML component must have:

1. **Unit tests**: Test feature calculation logic
2. **Integration tests**: Test model loading and inference
3. **Performance tests**: Measure inference latency
4. **Accuracy tests**: Compare predictions with labeled test set

```go
func TestAnomalyDetector(t *testing.T) {
    detector, err := anomaly.NewDetector("../../models/anomaly_detector_v1.0.0.onnx")
    require.NoError(t, err)
    defer detector.Close()

    t.Run("Normal conditions", func(t *testing.T) {
        features := &anomaly.Features{
            VolumeDeviation: 0.5,
            PriceChange: 0.02,
        }
        result, err := detector.Detect(features)
        require.NoError(t, err)
        assert.False(t, result.IsAnomaly)
        assert.Less(t, result.Score, 0.5)
    })

    t.Run("Anomalous conditions", func(t *testing.T) {
        features := &anomaly.Features{
            VolumeDeviation: 5.0,  // 5 sigma spike
            PriceChange: 0.15,     // 15% move
        }
        result, err := detector.Detect(features)
        require.NoError(t, err)
        assert.True(t, result.IsAnomaly)
        assert.Greater(t, result.Score, 0.8)
    })
}
```

---

## Training Pipeline

### Data Collection

#### ClickHouse Schema for ML Features

```sql
-- migrations/clickhouse/020_ml_features.up.sql
CREATE TABLE IF NOT EXISTS ml_features (
    timestamp DateTime,
    symbol String,

    -- Volatility features
    atr_14 Float64,
    atr_pct Float64,
    bb_width Float64,
    historical_vol Float64,

    -- Trend features
    ema_20 Float64,
    ema_50 Float64,
    ema_200 Float64,
    adx Float64,

    -- Volume features
    volume_24h Float64,
    volume_change_pct Float64,

    -- Structure features
    support_distance Float64,
    resistance_distance Float64,

    -- Derivatives features
    funding_rate Float64,
    liquidations_24h Float64,

    -- Target labels (for supervised learning)
    future_return_1h Float64,    -- Actual return 1h later
    future_return_24h Float64,   -- Actual return 24h later
    regime_label String          -- Manually labeled regime

) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (symbol, timestamp);
```

#### Feature Extraction Worker

```go
// internal/workers/ml/feature_extractor.go
type FeatureExtractorWorker struct {
    marketRepo    market_data.Repository
    clickhouse    *clickhouse.Client
    indicators    *indicators.Calculator
}

func (w *FeatureExtractorWorker) Run(ctx context.Context) error {
    // Run every 1 hour
    ticker := time.NewTicker(1 * time.Hour)

    for {
        select {
        case <-ticker.C:
            for _, symbol := range w.config.Symbols {
                features := w.extractFeatures(symbol)
                w.clickhouse.Insert("ml_features", features)
            }
        case <-ctx.Done():
            return nil
        }
    }
}
```

### Automated Retraining

```yaml
# airflow/dags/ml_retraining.py
from airflow import DAG
from airflow.operators.bash import BashOperator
from datetime import timedelta

dag = DAG(
    'ml_retraining',
    schedule_interval='@weekly',  # Retrain every week
    default_args={
        'retries': 2,
        'retry_delay': timedelta(minutes=5),
    }
)

# Task 1: Extract features from ClickHouse
extract_features = BashOperator(
    task_id='extract_features',
    bash_command='python /scripts/ml/extract_features.py',
    dag=dag
)

# Task 2: Train regime detector
train_regime = BashOperator(
    task_id='train_regime_detector',
    bash_command='python /scripts/ml/regime/train_model.py',
    dag=dag
)

# Task 3: Train anomaly detector
train_anomaly = BashOperator(
    task_id='train_anomaly_detector',
    bash_command='python /scripts/ml/anomaly/train_model.py',
    dag=dag
)

# Task 4: Validate models
validate_models = BashOperator(
    task_id='validate_models',
    bash_command='python /scripts/ml/validate_all.py',
    dag=dag
)

# Task 5: Deploy if validation passes
deploy_models = BashOperator(
    task_id='deploy_models',
    bash_command='bash /scripts/ml/deploy.sh',
    dag=dag
)

extract_features >> [train_regime, train_anomaly] >> validate_models >> deploy_models
```

---

## Model Monitoring

### Metrics to Track

Store in ClickHouse for analysis:

```sql
CREATE TABLE IF NOT EXISTS ml_predictions (
    timestamp DateTime,
    model_name String,
    model_version String,
    symbol String,

    -- Input features (for debugging)
    features String,  -- JSON

    -- Predictions
    prediction Float64,
    confidence Float64,

    -- Ground truth (filled later)
    actual_value Nullable(Float64),

    -- Performance
    inference_latency_ms Float64,
    error Nullable(Float64)  -- |prediction - actual|

) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (model_name, timestamp);
```

### Monitoring Queries

```sql
-- Model accuracy over time
SELECT
    toStartOfDay(timestamp) as day,
    model_name,
    avg(abs(prediction - actual_value)) as mae,
    sqrt(avg(pow(prediction - actual_value, 2))) as rmse
FROM ml_predictions
WHERE actual_value IS NOT NULL
    AND timestamp > now() - 30
GROUP BY day, model_name
ORDER BY day DESC;

-- Inference latency
SELECT
    model_name,
    avg(inference_latency_ms) as avg_latency,
    quantile(0.95)(inference_latency_ms) as p95_latency,
    quantile(0.99)(inference_latency_ms) as p99_latency
FROM ml_predictions
WHERE timestamp > now() - 7
GROUP BY model_name;

-- Prediction distribution drift
SELECT
    toStartOfWeek(timestamp) as week,
    avg(prediction) as mean_prediction,
    stddevPop(prediction) as std_prediction
FROM ml_predictions
WHERE model_name = 'regime_detector'
GROUP BY week
ORDER BY week DESC;
```

### Alerting Rules

```go
// internal/workers/monitoring/ml_monitor.go
type MLMonitorWorker struct {
    clickhouse *clickhouse.Client
    alerter    *alerting.Service
}

func (w *MLMonitorWorker) CheckModelHealth() error {
    // Alert 1: High error rate
    mae := w.clickhouse.Query("SELECT avg(error) FROM ml_predictions WHERE timestamp > now() - 1h")
    if mae > w.config.MAEThreshold {
        w.alerter.Send("ML model degraded: MAE = %.2f", mae)
    }

    // Alert 2: High latency
    p99 := w.clickhouse.Query("SELECT quantile(0.99)(inference_latency_ms) FROM ml_predictions WHERE timestamp > now() - 1h")
    if p99 > w.config.LatencyThreshold {
        w.alerter.Send("ML inference slow: P99 = %.0fms", p99)
    }

    // Alert 3: Prediction drift
    recentMean := w.clickhouse.Query("SELECT avg(prediction) FROM ml_predictions WHERE timestamp > now() - 1d")
    historicalMean := w.clickhouse.Query("SELECT avg(prediction) FROM ml_predictions WHERE timestamp BETWEEN now() - 30d AND now() - 7d")

    drift := abs(recentMean - historicalMean) / historicalMean
    if drift > 0.3 {  // 30% drift
        w.alerter.Send("ML prediction drift detected: %.1f%%", drift*100)
    }

    return nil
}
```

---

## Roadmap

### Phase 4 (Current) ✅

- [x] ONNX runtime infrastructure
- [x] Regime detector (Random Forest)
- [x] Feature extraction worker
- [x] Basic training pipeline

### Phase 5 (Q1 2025)

- [ ] Anomaly detection (Isolation Forest)
- [ ] Volatility forecasting (GARCH)
- [ ] Trade quality classifier
- [ ] ML monitoring dashboard
- [ ] Automated retraining pipeline

### Phase 6 (Q2 2025)

- [ ] Correlation regime detector
- [ ] Liquidity predictor
- [ ] Pattern recognition (CNN)
- [ ] A/B testing framework for models

### Phase 7 (Q3 2025+)

- [ ] Optimal execution (RL)
- [ ] Sentiment analysis (BERT)
- [ ] Multi-model ensemble
- [ ] AutoML for hyperparameter tuning

---

## Best Practices

### 1. Fail-Safe Design

- Always have rule-based fallback if ML model unavailable
- Circuit breaker pattern for model failures
- Graceful degradation

```go
func (e *Engine) GetRegime(symbol string) string {
    // Try ML model first
    if e.regimeClassifier != nil {
        result, err := e.regimeClassifier.Classify(features)
        if err == nil && result.Confidence > 0.6 {
            return result.Regime
        }
    }

    // Fallback to rule-based
    return e.ruleBasedRegimeDetector(symbol)
}
```

### 2. Explainability

- Log feature importance
- Store predictions with input features
- Provide reasoning for ML decisions

```go
type PredictionResult struct {
    Prediction      float64
    Confidence      float64
    FeatureImportance map[string]float64  // NEW
    Reasoning       string                 // NEW
}

func (c *Classifier) Explain(result *PredictionResult) string {
    // "Regime classified as 'trend_up' because:
    //  - ADX = 35 (strong trend)
    //  - EMA20 > EMA50 (bullish alignment)
    //  - Volume increasing (confirmation)"
}
```

### 3. Data Quality

- Validate input features before inference
- Handle missing values gracefully
- Detect and reject out-of-distribution inputs

```go
func (d *Detector) ValidateFeatures(features *Features) error {
    if features.ATR14 < 0 || features.ATR14 > 10000 {
        return fmt.Errorf("ATR out of range: %.2f", features.ATR14)
    }

    if math.IsNaN(features.ADX) {
        return fmt.Errorf("ADX is NaN")
    }

    return nil
}
```

### 4. Performance

- Batch predictions when possible
- Cache frequently used predictions
- Monitor inference latency

```go
// Batch inference for multiple symbols
func (d *Detector) DetectBatch(symbolFeatures map[string]*Features) map[string]*Result {
    results := make(map[string]*Result)

    // Prepare batch input tensor
    batchInput := make([][]float32, 0, len(symbolFeatures))
    symbols := make([]string, 0, len(symbolFeatures))

    for symbol, features := range symbolFeatures {
        batchInput = append(batchInput, features.ToTensor())
        symbols = append(symbols, symbol)
    }

    // Single inference call for all
    outputs := d.runtime.RunBatch(batchInput)

    // Map outputs back to symbols
    for i, symbol := range symbols {
        results[symbol] = ParseOutput(outputs[i])
    }

    return results
}
```

---

## Questions?

For ML-related questions or contributions:

1. Review existing implementations in `internal/ml/regime/`
2. Follow ONNX export standards
3. Add comprehensive tests
4. Document feature engineering
5. Monitor model performance

See also:

- `docs/AGENT_ARCHITECTURE.md` - How agents use ML tools
- `docs/specs.md` - System specifications
- `internal/ml/README.md` - ML infrastructure details
