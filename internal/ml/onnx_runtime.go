package ml

import (
	"fmt"

	onnxruntime "github.com/yalue/onnxruntime_go"

	"prometheus/pkg/errors"
)

// ONNXModel wraps ONNX Runtime session for ML inference
type ONNXModel struct {
	session     *onnxruntime.DynamicAdvancedSession
	inputName   string
	outputNames []string
}

// LoadONNXModel loads an ONNX model from file
func LoadONNXModel(modelPath string) (*ONNXModel, error) {
	// Initialize ONNX runtime environment (only once)
	err := onnxruntime.InitializeEnvironment()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize ONNX runtime")
	}

	// Create session options
	options, err := onnxruntime.NewSessionOptions()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create session options")
	}
	defer options.Destroy()

	// Load model with dynamic session (allows runtime tensor creation)
	// Input: "input" (feature vector)
	// Outputs: "output" (predicted class), "probabilities" (class probabilities)
	session, err := onnxruntime.NewDynamicAdvancedSession(modelPath,
		[]string{"input"}, []string{"output", "probabilities"}, options)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ONNX model")
	}

	return &ONNXModel{
		session:     session,
		inputName:   "input",
		outputNames: []string{"output", "probabilities"},
	}, nil
}

// Predict runs inference on the model with given features
// Returns predicted class name and probability map
func (m *ONNXModel) Predict(features []float64) (string, map[string]float64, error) {
	if m.session == nil {
		return "", nil, errors.New("model session is nil")
	}

	// Prepare input tensor: shape [1, num_features]
	inputShape := onnxruntime.NewShape(1, int64(len(features)))
	inputTensor, err := onnxruntime.NewTensor(inputShape, features)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create input tensor")
	}
	defer inputTensor.Destroy()

	// Prepare output tensors (pre-allocate)
	// Output 1: predicted class (int64, shape [1])
	classOutput := make([]int64, 1)
	classShape := onnxruntime.NewShape(1)
	classTensor, err := onnxruntime.NewTensor(classShape, classOutput)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create class output tensor")
	}
	defer classTensor.Destroy()

	// Output 2: probabilities (float64, shape [1, num_classes])
	numClasses := 7 // trend_up, trend_down, range, breakout, volatile, accumulation, distribution
	probabilitiesOutput := make([]float64, numClasses)
	probShape := onnxruntime.NewShape(1, int64(numClasses))
	probTensor, err := onnxruntime.NewTensor(probShape, probabilitiesOutput)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create probabilities output tensor")
	}
	defer probTensor.Destroy()

	// Run inference
	inputs := []onnxruntime.Value{inputTensor}
	outputs := []onnxruntime.Value{classTensor, probTensor}
	err = m.session.Run(inputs, outputs)
	if err != nil {
		return "", nil, errors.Wrap(err, "inference failed")
	}

	// Extract results from output tensors
	predictedClass := int(classOutput[0])

	// Map class index to name
	classNames := []string{"trend_up", "trend_down", "range", "breakout", "volatile", "accumulation", "distribution"}
	if predictedClass < 0 || predictedClass >= len(classNames) {
		return "", nil, fmt.Errorf("invalid class index: %d", predictedClass)
	}

	predictedClassName := classNames[predictedClass]

	// Build probability map
	probMap := make(map[string]float64)
	for i, prob := range probabilitiesOutput {
		if i < len(classNames) {
			probMap[classNames[i]] = prob
		}
	}

	return predictedClassName, probMap, nil
}

// Destroy cleans up the ONNX session
func (m *ONNXModel) Destroy() {
	if m.session != nil {
		m.session.Destroy()
		m.session = nil
	}
}

