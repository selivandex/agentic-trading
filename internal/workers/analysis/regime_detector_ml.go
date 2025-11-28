package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/regime"
	"prometheus/internal/events"
	regimeml "prometheus/internal/ml/regime"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// RegimeDetectorML performs ML-based regime detection with LLM interpretation
// This is the Phase 4 enhanced version using ONNX model + RegimeInterpreter agent
type RegimeDetectorML struct {
	*workers.BaseWorker
	mdRepo           market_data.Repository
	regimeRepo       regime.Repository
	classifier       *regimeml.Classifier // ML model for classification
	interpreterAgent agent.Agent          // LLM agent for interpretation (from factory)
	runner           *runner.Runner       // ADK runner for interpreter agent
	sessionService   session.Service      // Session service
	kafka            *kafka.Producer
	eventPublisher   *events.WorkerPublisher
	symbols          []string
}

// NewRegimeDetectorML creates a new ML-based regime detector worker
func NewRegimeDetectorML(
	mdRepo market_data.Repository,
	regimeRepo regime.Repository,
	classifier *regimeml.Classifier,
	interpreterAgent agent.Agent, // Pre-configured agent from factory
	sessionService session.Service,
	kafka *kafka.Producer,
	symbols []string,
	interval time.Duration,
	enabled bool,
) (*RegimeDetectorML, error) {
	var runnerInstance *runner.Runner

	// Create ADK runner if agent is provided
	if interpreterAgent != nil && sessionService != nil {
		var err error
		runnerInstance, err = runner.New(runner.Config{
			AppName:        "regime_interpreter",
			Agent:          interpreterAgent,
			SessionService: sessionService,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create ADK runner for interpreter")
		}
	}

	return &RegimeDetectorML{
		BaseWorker:       workers.NewBaseWorker("regime_detector_ml", interval, enabled),
		mdRepo:           mdRepo,
		regimeRepo:       regimeRepo,
		classifier:       classifier,
		interpreterAgent: interpreterAgent,
		runner:           runnerInstance,
		sessionService:   sessionService,
		kafka:            kafka,
		eventPublisher:   events.NewWorkerPublisher(kafka),
		symbols:          symbols,
	}, nil
}

// Run executes one iteration of ML-based regime detection
func (rd *RegimeDetectorML) Run(ctx context.Context) error {
	rd.Log().Debug("Regime detector (ML): starting iteration")

	if rd.classifier == nil {
		rd.Log().Warn("ML classifier not loaded, skipping regime detection")
		return nil
	}

	if len(rd.symbols) == 0 {
		rd.Log().Warn("No symbols configured for regime detection")
		return nil
	}

	for _, symbol := range rd.symbols {
		if err := rd.detectRegime(ctx, symbol); err != nil {
			rd.Log().Error("Failed to detect regime",
				"symbol", symbol,
				"error", err,
			)
			continue
		}
	}

	rd.Log().Debug("Regime detection complete")
	return nil
}

// detectRegime performs ML-based regime detection for a single symbol
func (rd *RegimeDetectorML) detectRegime(ctx context.Context, symbol string) error {
	// 1. Get latest features from ClickHouse
	features, err := rd.regimeRepo.GetLatestFeatures(ctx, symbol)
	if err != nil {
		return errors.Wrap(err, "failed to get features")
	}

	// 2. Run ML classification
	classification, err := rd.classifier.Classify(features)
	if err != nil {
		return errors.Wrap(err, "ML classification failed")
	}

	rd.Log().Info("Regime classified",
		"symbol", symbol,
		"regime", classification.Regime,
		"confidence", classification.Confidence,
		"probabilities", classification.Probabilities,
	)

	// Skip if confidence too low
	if classification.Confidence < 0.6 {
		rd.Log().Warn("Low confidence, skipping regime update",
			"symbol", symbol,
			"confidence", classification.Confidence,
		)
		return nil
	}

	// 3. Get previous regime
	previousRegime, _ := rd.regimeRepo.GetLatest(ctx, symbol)

	// 4. Only proceed if regime changed or first detection
	if previousRegime != nil && previousRegime.Regime == classification.Regime {
		rd.Log().Debug("Regime unchanged", "symbol", symbol, "regime", classification.Regime)
		return nil
	}

	// 5. LLM Interpretation: Call RegimeInterpreter agent
	interpretation, err := rd.interpretRegime(ctx, symbol, classification, features)
	if err != nil {
		rd.Log().Error("Failed to interpret regime", "error", err)
		// Continue anyway with empty interpretation
		interpretation = &RegimeInterpretation{}
	}

	// 6. Store regime
	newRegime := &regime.MarketRegime{
		Symbol:       symbol,
		Timestamp:    time.Now(),
		Regime:       classification.Regime,
		Confidence:   classification.Confidence,
		Volatility:   classifyVolatility(features.ATRPct),
		Trend:        classifyTrend(features),
		ATR14:        features.ATR14,
		ADX:          features.ADX,
		BBWidth:      features.BBWidth,
		Volume24h:    features.Volume24h,
		VolumeChange: features.VolumeChangePct,
	}

	if err := rd.regimeRepo.Store(ctx, newRegime); err != nil {
		return errors.Wrap(err, "failed to store regime")
	}

	// 7. Publish regime_changed event
	rd.publishRegimeChangeEvent(ctx, symbol, previousRegime, newRegime, interpretation)

	oldRegimeStr := "none"
	if previousRegime != nil {
		oldRegimeStr = string(previousRegime.Regime)
	}

	rd.Log().Info("Regime updated",
		"symbol", symbol,
		"old_regime", oldRegimeStr,
		"new_regime", newRegime.Regime,
		"confidence", newRegime.Confidence,
	)

	return nil
}

// RegimeInterpretation holds strategic recommendations from LLM
type RegimeInterpretation struct {
	Explanation            string
	StrategicGuidance      string
	PositionSizeMultiplier float64
	CashReserveTarget      float64
	FavoredStrategies      []string
	AvoidedStrategies      []string
	RiskAdjustments        map[string]interface{}
}

// interpretRegime calls LLM agent to interpret the classified regime
func (rd *RegimeDetectorML) interpretRegime(
	ctx context.Context,
	symbol string,
	classification *regimeml.ClassificationResult,
	features *regime.Features,
) (*RegimeInterpretation, error) {
	if rd.runner == nil {
		// Return default interpretation if runner not available
		return &RegimeInterpretation{
			Explanation:            fmt.Sprintf("Regime %s detected with %.1f%% confidence (LLM interpretation unavailable)", classification.Regime, classification.Confidence*100),
			StrategicGuidance:      "Operating with default parameters",
			PositionSizeMultiplier: 1.0,
			CashReserveTarget:      10.0,
			FavoredStrategies:      []string{},
			AvoidedStrategies:      []string{},
			RiskAdjustments:        make(map[string]interface{}),
		}, nil
	}

	// Build input data for agent
	// Agent's system prompt template (agents/regime_interpreter) will be rendered automatically
	// We just need to provide the data in user message
	promptData := map[string]interface{}{
		"Symbol":        symbol,
		"Regime":        string(classification.Regime),
		"Confidence":    classification.Confidence,
		"Probabilities": classification.Probabilities,
		"Features":      features,
	}

	// Convert to JSON to pass as user input
	promptJSON, err := json.Marshal(promptData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal input data")
	}

	// Build user message - agent will process this with its configured template
	input := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: fmt.Sprintf("Analyze this regime classification:\n%s", string(promptJSON))},
		},
	}

	// Run agent with timeout
	agentCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sessionID := uuid.New().String()
	userID := "system" // System-level analysis

	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeNone,
	}

	var responseText string
	for event, err := range rd.runner.Run(agentCtx, userID, sessionID, input, runConfig) {
		if err != nil {
			return nil, errors.Wrap(err, "agent run failed")
		}

		if event == nil || event.LLMResponse.Partial {
			continue
		}

		// Extract response text
		if event.LLMResponse.Content != nil {
			for _, part := range event.LLMResponse.Content.Parts {
				if part.Text != "" {
					responseText = part.Text
				}
			}
		}

		// Check if complete
		if event.TurnComplete && event.IsFinalResponse() {
			break
		}
	}

	if responseText == "" {
		return nil, errors.New("no response from agent")
	}

	// Parse JSON response from agent
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &output); err != nil {
		rd.Log().Warn("Failed to parse agent output as JSON, using raw text", "error", err)
		// Fallback to basic interpretation
		return &RegimeInterpretation{
			Explanation:            responseText,
			StrategicGuidance:      responseText,
			PositionSizeMultiplier: 1.0,
			CashReserveTarget:      10.0,
			FavoredStrategies:      []string{},
			AvoidedStrategies:      []string{},
			RiskAdjustments:        make(map[string]interface{}),
		}, nil
	}

	interpretation := &RegimeInterpretation{
		Explanation:            getString(output, "explanation"),
		StrategicGuidance:      getString(output, "strategic_guidance"),
		PositionSizeMultiplier: getFloat(output, "position_size_multiplier", 1.0),
		CashReserveTarget:      getFloat(output, "cash_reserve_target", 10.0),
		FavoredStrategies:      getStringSlice(output, "favored_strategies"),
		AvoidedStrategies:      getStringSlice(output, "avoided_strategies"),
		RiskAdjustments:        getMap(output, "risk_adjustments"),
	}

	return interpretation, nil
}

// publishRegimeChangeEvent publishes regime change to Kafka
func (rd *RegimeDetectorML) publishRegimeChangeEvent(
	ctx context.Context,
	symbol string,
	previousRegime *regime.MarketRegime,
	newRegime *regime.MarketRegime,
	interpretation *RegimeInterpretation,
) {
	oldRegimeStr := ""
	if previousRegime != nil {
		oldRegimeStr = string(previousRegime.Regime)
	}

	// Calculate volatility as percentage (simplified)
	volatilityFloat := 0.0
	switch newRegime.Volatility {
	case regime.VolLow:
		volatilityFloat = 1.0
	case regime.VolMedium:
		volatilityFloat = 2.5
	case regime.VolHigh:
		volatilityFloat = 4.0
	case regime.VolExtreme:
		volatilityFloat = 6.0
	}

	// Publish using WorkerPublisher
	if err := rd.eventPublisher.PublishRegimeChangeML(
		ctx,
		symbol,
		oldRegimeStr,
		string(newRegime.Regime),
		string(newRegime.Trend),
		newRegime.Confidence,
		volatilityFloat,
		interpretation.Explanation,
		interpretation.StrategicGuidance,
		interpretation.PositionSizeMultiplier,
		interpretation.CashReserveTarget,
		interpretation.FavoredStrategies,
		interpretation.AvoidedStrategies,
	); err != nil {
		rd.Log().Error("Failed to publish regime_changed event", "error", err)
	}
}

// Helper functions for regime classification

func classifyVolatility(atrPct float64) regime.VolLevel {
	if atrPct > 5.0 {
		return regime.VolExtreme
	} else if atrPct > 3.0 {
		return regime.VolHigh
	} else if atrPct > 1.5 {
		return regime.VolMedium
	}
	return regime.VolLow
}

func classifyTrend(features *regime.Features) regime.TrendType {
	alignment := features.EMAAlignment
	switch alignment {
	case "strong_bullish", "bullish":
		return regime.TrendBullish
	case "strong_bearish", "bearish":
		return regime.TrendBearish
	}
	return regime.TrendNeutral
}

// Helper functions for parsing agent output

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]interface{}, key string, defaultVal float64) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return defaultVal
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok {
		if slice, ok := v.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
		if slice, ok := v.([]string); ok {
			return slice
		}
	}
	return []string{}
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if mapVal, ok := v.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return make(map[string]interface{})
}
