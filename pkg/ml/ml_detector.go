package ml

import (
	"fmt"
	"macnet/pkg/signal"
)

const (
	// DefaultNumFeatures is the total count of features in the vector.
	// 1 (Lang) + 10 (Heuristics) + 4 (Structural) + 6 (Stats) = 21
	DefaultNumFeatures = 21

	// DefaultMLWeight represents the influence of the ML score in the final calculation.
	// A value of 0.7 means the final score is 70% ML and 30% heuristic signals.
	DefaultMLWeight = 0.7

	// DefaultMinConfidence is the threshold used to determine if a file is
	// flagged as AI-generated when ML results are available.
	DefaultMinConfidence = 0.5
)

type MLDetector struct {
	onnxDetector     *ONNXDetector
	featureExtractor *FeatureExtractor
	config           DetectionConfig
}

type DetectionConfig struct {
	ModelPath     string
	MLWeight      float64
	FallbackMode  bool
	RequireML     bool
	MinConfidence float64
	NumFeatures   int
}

func NewMLDetector(config DetectionConfig) (*MLDetector, error) {
	if config.MLWeight == 0.0 {
		config.MLWeight = DefaultMLWeight
	}
	if config.MinConfidence == 0.0 {
		config.MinConfidence = DefaultMinConfidence
	}
	if config.NumFeatures == 0 {
		config.NumFeatures = DefaultNumFeatures
	}

	featureExtractor := NewFeatureExtractor()

	var onnxDetector *ONNXDetector
	var err error

	if config.ModelPath != "" {
		modelConfig := ModelConfig{
			ModelPath:      config.ModelPath,
			InputName:      "input",
			OutputName:     "output",
			NumFeatures:    config.NumFeatures,
			OptimizeMemory: true,
		}

		onnxDetector, err = NewONNXDetector(modelConfig)
		if err != nil {
			if config.RequireML {
				return nil, fmt.Errorf("ML model required but failed to load: %w", err)
			}
			onnxDetector = &ONNXDetector{initialized: false}
		}
	} else {
		onnxDetector = &ONNXDetector{initialized: false}
	}

	return &MLDetector{
		onnxDetector:     onnxDetector,
		featureExtractor: featureExtractor,
		config:           config,
	}, nil
}

type DetectionResult struct {
	FinalScore     float64
	HeuristicScore float64
	MLScore        float64
	UsedML         bool
	Confidence     float64
	IsHighRisk     bool
}

func (m *MLDetector) Detect(signals []signal.Signal, language string, code string) (DetectionResult, error) {
	if m.onnxDetector == nil || !m.onnxDetector.initialized {
		hScore := calculateHeuristicMaxScore(signals)
		return DetectionResult{
			FinalScore:     hScore,
			HeuristicScore: hScore,
			UsedML:         false,
		}, nil
	}

	metrics := ExtractCodeMetrics(code)
	vector := m.featureExtractor.Extract(signals, language, metrics)

	mlScore, err := m.onnxDetector.Predict(vector.Features)
	if err != nil {
		if m.config.FallbackMode {
			hScore := calculateHeuristicMaxScore(signals)
			return DetectionResult{FinalScore: hScore, HeuristicScore: hScore, UsedML: false}, nil
		}
		return DetectionResult{}, err
	}

	heuristicScore := calculateHeuristicMaxScore(signals)
	mlScore64 := float64(mlScore)
	finalScore := (mlScore64 * m.config.MLWeight) + (heuristicScore * (1.0 - m.config.MLWeight))

	return DetectionResult{
		MLScore:        mlScore64,
		HeuristicScore: heuristicScore,
		FinalScore:     finalScore,
		Confidence:     0.7,
		UsedML:         true,
		IsHighRisk:     finalScore >= m.config.MinConfidence,
	}, nil
}

func calculateHeuristicMaxScore(signals []signal.Signal) float64 {
	if len(signals) == 0 {
		return 0.0
	}

	maxScore := 0.0
	for _, signal := range signals {
		if signal.Score > maxScore {
			maxScore = signal.Score
		}
	}
	return maxScore
}

func (d *MLDetector) GetModelInfo() map[string]interface{} {
	info := map[string]interface{}{
		"ml_enabled":     d.onnxDetector.IsInitialized(),
		"ml_weight":      d.config.MLWeight,
		"min_confidence": d.config.MinConfidence,
		"fallback_mode":  d.config.FallbackMode,
	}
	if d.onnxDetector.IsInitialized() {
		modelInfo := d.onnxDetector.GetModelInfo()
		for k, v := range modelInfo {
			info[k] = v
		}
	}
	return info
}

func (d *MLDetector) Close() error {
	if d.onnxDetector != nil {
		return d.onnxDetector.Close()
	}
	return nil
}
