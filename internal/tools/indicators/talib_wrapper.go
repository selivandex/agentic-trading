package indicators

import (
	"prometheus/internal/domain/market_data"
	"prometheus/pkg/errors"
)

// TalibData holds OHLCV data in format expected by ta-lib
type TalibData struct {
	Open   []float64
	High   []float64
	Low    []float64
	Close  []float64
	Volume []float64
}

// PrepareData converts domain OHLCV candles to ta-lib format
// Returns separate slices for each price component
func PrepareData(candles []market_data.OHLCV) (*TalibData, error) {
	if len(candles) == 0 {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "no candles provided")
	}
	data := &TalibData{
		Open:   make([]float64, len(candles)),
		High:   make([]float64, len(candles)),
		Low:    make([]float64, len(candles)),
		Close:  make([]float64, len(candles)),
		Volume: make([]float64, len(candles)),
	}
	// Fill arrays (ta-lib expects chronological order - oldest first)
	// Reverse if needed (our candles are DESC by default)
	for i, candle := range candles {
		idx := len(candles) - 1 - i // Reverse index
		data.Open[idx] = candle.Open
		data.High[idx] = candle.High
		data.Low[idx] = candle.Low
		data.Close[idx] = candle.Close
		data.Volume[idx] = candle.Volume
	}
	return data, nil
}

// PrepareCloses extracts only close prices (for simple indicators like RSI, EMA)
func PrepareCloses(candles []market_data.OHLCV) ([]float64, error) {
	if len(candles) == 0 {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "no candles provided")
	}
	closes := make([]float64, len(candles))
	// Reverse to chronological order (oldest first)
	for i, candle := range candles {
		idx := len(candles) - 1 - i
		closes[idx] = candle.Close
	}
	return closes, nil
}

// GetLastValue returns the most recent value from ta-lib output
// ta-lib returns full array, we typically only need the latest value
func GetLastValue(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0, errors.Wrapf(errors.ErrInternal, "no values returned from indicator")
	}
	// Last value in array is the most recent
	return values[len(values)-1], nil
}

// GetLastNValues returns last N values from ta-lib output
func GetLastNValues(values []float64, n int) ([]float64, error) {
	if len(values) == 0 {
		return nil, errors.Wrapf(errors.ErrInternal, "no values returned from indicator")
	}
	if n <= 0 || n > len(values) {
		n = len(values)
	}
	// Return last N values
	start := len(values) - n
	return values[start:], nil
}

// ValidateMinLength checks if we have enough data for indicator calculation
func ValidateMinLength(candles []market_data.OHLCV, minLength int, indicatorName string) error {
	if len(candles) < minLength {
		return errors.Wrapf(errors.ErrInvalidInput,
			"%s requires at least %d candles, got %d",
			indicatorName, minLength, len(candles))
	}
	return nil
}
