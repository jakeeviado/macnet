package detector

import (
	"fmt"
	"os"

	"macnet/pkg/analyzer"
	"macnet/pkg/ml"
)

type Detector struct {
	detectorConfig DetectorConfiguration
	analyzer       *analyzer.Analyzer
	mlDetector     *ml.MLDetector
}

// Initializes and returns a new Detector instance with the provided configuration,
// loading the ONNX model if machine learning is enabled.
func New(config DetectorConfiguration) (*Detector, error) {
	d := &Detector{
		detectorConfig: config,
		analyzer:       analyzer.New(config.Profile),
	}

	if config.IsMLOnly && (!config.UseML || config.ModelPath == "") {
		return nil, fmt.Errorf("ML-only mode requires a model, but none was configured")
	}
	if !config.UseML || config.ModelPath == "" {
		return d, nil
	}

	mlDetector, err := ml.NewMLDetector(ml.DetectionConfig{
		ModelPath:     config.ModelPath,
		MLWeight:      config.MLWeight,
		FallbackMode:  !config.IsMLOnly,
		RequireML:     config.IsMLOnly,
		MinConfidence: config.Threshold,
		NumFeatures:   ml.NumFeatures,
	})
	if err != nil {
		if config.IsMLOnly {
			return nil, fmt.Errorf("ML-only mode: failed to load model %s: %w", config.ModelPath, err)
		}
		fmt.Fprintf(os.Stderr, "Warning: ML initialization failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "Falling back to heuristics-only mode\n")
		return d, nil
	}

	d.mlDetector = mlDetector
	if config.IsVerbose {
		fmt.Fprintf(os.Stderr, "ML model loaded: %s\n", config.ModelPath)
	}
	return d, nil
}
