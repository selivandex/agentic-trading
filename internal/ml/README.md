# ML Infrastructure (Phase 4)

Machine Learning infrastructure for Prometheus trading system.

## Components

### ONNX Runtime Wrapper
- **File**: `onnx_runtime.go`
- **Purpose**: Load and run ONNX models in Go
- **Features**: 
  - Dynamic tensor creation
  - Multi-output support (class + probabilities)
  - Automatic cleanup

### Regime Classifier
- **Package**: `regime/`
- **Model**: Random Forest / XGBoost (trained offline, exported to ONNX)
- **Input**: 22 market features (volatility, trend, volume, structure, derivatives)
- **Output**: Regime classification + confidence + probabilities

## Usage

### Loading Model

```go
classifier, err := regime.NewClassifier("models/regime_detector.onnx")
if err != nil {
    log.Fatal(err)
}
defer classifier.Close()
```

### Running Inference

```go
features := &regime.Features{
    ATR14:  450.0,
    ATRPct: 2.5,
    ADX:    35.0,
    // ... fill other features
}

result, err := classifier.Classify(features)
// result.Regime = "trend_up"
// result.Confidence = 0.85
// result.Probabilities = {"trend_up": 0.85, "range": 0.10, ...}
```

## Training Pipeline

See `scripts/ml/regime/` for training scripts:

1. **Collect Data**: Run `feature_extractor` worker for 2-4 weeks
2. **Label Data**: `python scripts/ml/regime/label_data.py`
3. **Train Model**: `python scripts/ml/regime/train_model.py`
4. **Deploy**: Copy `regime_detector.onnx` to `models/` directory

## Dependencies

- `github.com/yalue/onnxruntime_go` - ONNX Runtime for Go
- Python packages (see `scripts/ml/regime/requirements.txt`)

## Testing

```bash
go test ./internal/ml/regime/...
```

Note: Tests will skip if model file not present.




