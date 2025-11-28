package regime

import (
	"prometheus/internal/domain/regime"
	"prometheus/internal/ml"
	"prometheus/pkg/errors"
)

// Classifier performs ML-based regime classification using ONNX model
type Classifier struct {
	model *ml.ONNXModel
}

// NewClassifier creates a new regime classifier with loaded ONNX model
func NewClassifier(modelPath string) (*Classifier, error) {
	model, err := ml.LoadONNXModel(modelPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load regime model")
	}

	return &Classifier{model: model}, nil
}

// ClassificationResult contains the result of regime classification
type ClassificationResult struct {
	Regime        regime.RegimeType  // Predicted regime type
	Confidence    float64            // Confidence score (max probability)
	Probabilities map[string]float64 // Probability distribution over all classes
}

// Classify runs ML inference to classify the market regime
func (c *Classifier) Classify(features *regime.Features) (*ClassificationResult, error) {
	if c.model == nil {
		return nil, errors.New("classifier model is not loaded")
	}

	// Convert features to vector (22 features total)
	featureVector := features.ToFeatureVector()

	// Run inference
	predictedRegime, probabilities, err := c.model.Predict(featureVector)
	if err != nil {
		return nil, errors.Wrap(err, "classification failed")
	}

	// Find maximum probability (confidence)
	maxProb := 0.0
	for _, prob := range probabilities {
		if prob > maxProb {
			maxProb = prob
		}
	}

	return &ClassificationResult{
		Regime:        regime.RegimeType(predictedRegime),
		Confidence:    maxProb,
		Probabilities: probabilities,
	}, nil
}

// Close cleans up the classifier resources
func (c *Classifier) Close() {
	if c.model != nil {
		c.model.Destroy()
		c.model = nil
	}
}
