// Package analyzer provides tools for heuristic analysis of code to detect
// patterns that may indicate elevated review risk in codebases.
//
// # CRITICAL: ML PIPELINE SYNCHRONIZATION
//
// The signals produced by this package serve as the primary input features for
// the Machine Learning model.
//
// If you add a NEW detection signal here, you MUST:
//  1. Append the new signal name to the 'featureRegistry' in 'pkg/ml/feature_extractor.go'.
//  2. Update 'ml.DefaultNumFeatures' to reflect the new count.
//  3. Re-train the ONNX model to include the new signal.
//
// Failure to do this will result in the ML model ignoring the new heuristic,
// as the FeatureExtractor will not include it in the numerical vector passed
// to the inference engine.
//
// # IMPROVING AND MAINTAINING THE HEURISTIC ANALYSIS
//
// When adding a New Detection Signal:
//
// 1. Define the detection logic under analyzer/signals directory:
//
//    func CheckNewSignal(code string) models.Signal {
//        // implementation logic...
//        return models.Signal{
//            Name:        "new_signal", // This name MUST match the ml Registry
//            Score:       calculatedScore,
//            Description: "What this detects",
//        }
//    }
//
// 2. Register it in the Analyze method below:
//
//    signals = append(siglist, signals.CheckNewSignal(code))
//
// 3. Register it in pkg/ml/feature_extractor.go (FeatureRegistry) to ensure ML visibility.

package analyzer

import (
	"macnet/pkg/analyzer/signals"
	"macnet/pkg/models"
)

type Analyzer struct{}

func New() *Analyzer {
	return &Analyzer{}
}

// Analyze runs the full suite of heuristic checks against the provided code.
// It returns a slice of 'Signals', each representing a different detection metric.
func (a *Analyzer) AnalyzeFile(code string, language string) []models.Signal {
	sigList := make([]models.Signal, 0)
	sigList = append(sigList, signals.CheckGenericNaming(code))
	sigList = append(sigList, signals.CheckRepetitivePatterns(code))
	sigList = append(sigList, signals.CheckCodeComplexity(code))
	sigList = append(sigList, signals.CheckFormattingConsistency(code))
	sigList = append(sigList, signals.CheckCommentRedundancy(code))
	sigList = append(sigList, signals.CheckEmojiSentiment(code))
	sigList = append(sigList, signals.CheckIdentifierOrder(code))
	sigList = append(sigList, signals.CheckDefensiveRatio(code))
	return sigList
}
