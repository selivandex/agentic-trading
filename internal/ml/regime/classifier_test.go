package regime

import (
	"os"
	"testing"

	"prometheus/internal/domain/regime"
)

func TestClassifier_Classify(t *testing.T) {
	// Skip if model file doesn't exist
	modelPath := "../../../models/regime_detector.onnx"
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skip("Model file not found, skipping test. Train model first using scripts/ml/regime/train_model.py")
	}

	classifier, err := NewClassifier(modelPath)
	if err != nil {
		t.Fatalf("Failed to load classifier: %v", err)
	}
	defer classifier.Close()

	// Test with sample features (realistic BTC values)
	features := &regime.Features{
		Symbol: "BTC-USDT",
		// Volatility features
		ATR14:         450.0,
		ATRPct:        2.5,
		BBWidth:       4.2,
		HistoricalVol: 3.1,
		// Trend features
		ADX:              35.0,
		EMA9:             43500.0,
		EMA21:            43200.0,
		EMA55:            42800.0,
		EMA200:           42000.0,
		EMAAlignment:     "bullish",
		HigherHighsCount: 8,
		LowerLowsCount:   2,
		// Volume features
		Volume24h:             150000.0,
		VolumeChangePct:       15.0,
		VolumePriceDivergence: 0.65,
		// Structure features
		SupportBreaks:        2,
		ResistanceBreaks:     5,
		ConsolidationPeriods: 3,
		// Cross-asset features
		BTCDominance:         0.52,
		CorrelationTightness: 0.75,
		// Derivatives features
		FundingRate:     0.0001,
		FundingRateMA:   0.00012,
		Liquidations24h: 5000000.0,
	}

	result, err := classifier.Classify(features)
	if err != nil {
		t.Fatalf("Classification failed: %v", err)
	}

	// Validate results
	if result == nil {
		t.Fatal("Classification result is nil")
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Invalid confidence: %f (expected 0-1)", result.Confidence)
	}

	if result.Regime == "" {
		t.Error("Regime type is empty")
	}

	if !result.Regime.Valid() {
		t.Errorf("Invalid regime type: %s", result.Regime)
	}

	if len(result.Probabilities) == 0 {
		t.Error("No probabilities returned")
	}

	// Sum of probabilities should be ~1.0
	sum := 0.0
	for _, prob := range result.Probabilities {
		sum += prob
		if prob < 0 || prob > 1 {
			t.Errorf("Invalid probability value: %f", prob)
		}
	}

	// Allow small floating point error
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("Probabilities don't sum to 1.0: %f", sum)
	}

	t.Logf("Classification successful: regime=%s, confidence=%.2f%%", result.Regime, result.Confidence*100)
	t.Logf("Probabilities: %v", result.Probabilities)
}

func TestClassifier_CloseIdempotent(t *testing.T) {
	modelPath := "../../../models/regime_detector.onnx"
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skip("Model file not found")
	}

	classifier, err := NewClassifier(modelPath)
	if err != nil {
		t.Fatalf("Failed to load classifier: %v", err)
	}

	// Close multiple times should not panic
	classifier.Close()
	classifier.Close()
	classifier.Close()
}


