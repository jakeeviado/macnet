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
func New(config DetectorConfiguration) *Detector {
	detector := &Detector{
		detectorConfig: config,
		analyzer:       analyzer.New(),
	}

	if config.UseML && config.ModelPath != "" {
		mlConfig := ml.DetectionConfig{
			ModelPath:     config.ModelPath,
			MLWeight:      config.MLWeight,
			FallbackMode:  !config.IsMLOnly,
			RequireML:     config.IsMLOnly,
			MinConfidence: config.Threshold,
			NumFeatures:   ml.DefaultNumFeatures,
		}

		machineLearningDetector, err := ml.NewMLDetector(mlConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: ML initialization failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "Falling back to heuristics-only mode\n")
		} else {
			detector.mlDetector = machineLearningDetector
			if config.IsVerbose {
				fmt.Fprintf(os.Stderr, "ML model loaded: %s\n", config.ModelPath)
			}
		}
	}
	return detector
}
