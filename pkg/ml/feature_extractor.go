// Package ml provides the machine learning pipeline for MACnet, including
// feature extraction, statistical analysis of heuristic signals, and
// interfacing with ONNX-based inference engines.
//
// # INTEGRATION WITH ANALYZER
//
// This package acts as the bridge between the 'analyzer' package and the ML model.
// While the analyzer produces raw Signal scores (heuristics), the ml package
// transforms those signals into a fixed-length numerical vector (FeatureVector).
//
// MAINTENANCE & SYNCHRONIZATION
//
// When a new heuristic signal is added to the Analyzer (e.g., checkNewSignal),
// it MUST also be added to the FeatureRegistry within this package.
//
// Failure to sync the Analyzer and the ML Registry will result in the ML model
// ignoring the new signal entirely, as it will not be included in the
// feature vector passed to the ONNX session.
//
// # TRAINING WORKFLOW
//
// These heuristics are used for training by logging the FeatureVector
// along with a ground-truth label (AI versus Human) to a dataset.
// The model learns to associate specific heuristic score patterns—such as
// high comment density combined with low code complexity—to predict the
// likelihood of AI generation.
//
// # FEATURE SCHEME
//
// Features are extracted in a specific, immutable order to maintain compatibility
// with the underlying ONNX tensors:
//  1. Language Identification (integer mapping)
//  2. Heuristic Signal Scores (indexed via the FeatureRegistry)
//  3. Structural Code Metrics (line counts, density, etc.)
//  4. Statistical Aggregates (mean, variance, and distribution of signals)
//
// # MAINTENANCE
//
// The FeatureRegistry is an append-only structure. Reordering or deleting
// entries will shift feature indices and invalidate existing models.

package ml

import (
	"macnet/pkg/models"
	"math"
	"slices"
)

type FeatureVector struct {
	Features []float32
	Names    []string
}

type FeatureExtractor struct {
	languageIDMap   map[string]int
	featureRegistry []string
}

// NewFeatureExtractor initializes the feature extractor with a fixed schema.
//
// WARNING: The order and content of 'featureRegistry' are CRITICAL for ML model stability.
// Adding, removing, or reordering signals here will shift the feature indices and
// cause existing ONNX models to produce incorrect results.
func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{
		languageIDMap: map[string]int{
			"unknown":    0,
			"python":     1,
			"java":       2,
			"javascript": 3,
			"typescript": 4,
			"go":         5,
			"rust":       6,
			"cpp":        7,
			"c":          8,
			"ruby":       9,
			"php":        10,
			"csharp":     11,
			"kotlin":     12,
			"swift":      13,
		},
		featureRegistry: []string{
			"comment_density", "generic_naming", "repetitive_patterns",
			"code_complexity", "formatting_consistency",
			"comment_redundancy", "emoji_sentiment", "identifier_order", "defensive_ratio",
		},
	}
}

// This fucntion converts raw signals and metrics into a FeatureVector.
// It ensures that features are mapped to consistent indices defined in the featureRegistry.
func (fe *FeatureExtractor) Extract(signals []models.Signal, language string, metrics CodeMetrics) *FeatureVector {
	features := make([]float32, 0, 50)
	names := make([]string, 0, 50)

	features = append(features, float32(fe.languageIDMap[language]))
	names = append(names, "language_id")

	sigMap := make(map[string]float64)
	for _, s := range signals {
		sigMap[s.Name] = s.Score
	}

	for _, regName := range fe.featureRegistry {
		score := sigMap[regName]
		features = append(features, float32(score))
		names = append(names, regName)
	}

	features = append(features, float32(metrics.LineCount))
	names = append(names, "line_count")

	features = append(features, float32(metrics.CharCount))
	names = append(names, "char_count")

	features = append(features, float32(metrics.AvgLineLength))
	names = append(names, "avg_line_length")

	features = append(features, float32(metrics.EmptyLineRatio))
	names = append(names, "empty_line_ratio")

	signalScores := make([]float64, 0, len(signals))
	for _, sig := range signals {
		signalScores = append(signalScores, sig.Score)
	}

	stats := calculateStats(signalScores)

	features = append(features, float32(stats.Mean))
	names = append(names, "signal_mean")

	features = append(features, float32(stats.StdDev))
	names = append(names, "signal_stddev")

	features = append(features, float32(stats.Max))
	names = append(names, "signal_max")

	features = append(features, float32(stats.Min))
	names = append(names, "signal_min")

	features = append(features, float32(stats.Median))
	names = append(names, "signal_median")

	activeSignals := 0
	for _, sig := range signals {
		if sig.Score > 0.0 {
			activeSignals++
		}
	}
	features = append(features, float32(activeSignals))
	names = append(names, "active_signal_count")

	return &FeatureVector{
		Features: features,
		Names:    names,
	}
}

type CodeMetrics struct {
	LineCount      int
	CharCount      int
	AvgLineLength  float64
	EmptyLineRatio float64
}

type Stats struct {
	Mean   float64
	StdDev float64
	Max    float64
	Min    float64
	Median float64
}

func calculateStats(scores []float64) Stats {
	if len(scores) == 0 {
		return Stats{}
	}

	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	mean := sum / float64(len(scores))

	variance := 0.0
	for _, s := range scores {
		diff := s - mean
		variance += diff * diff
	}
	variance /= float64(len(scores))
	stdDev := math.Sqrt(variance)

	max := slices.Max(scores)
	min := slices.Min(scores)

	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	slices.Sort(sorted)

	var median float64
	n := len(sorted)
	if n%2 == 0 {
		median = (sorted[n/2-1] + sorted[n/2]) / 2.0
	} else {
		median = sorted[n/2]
	}

	return Stats{
		Mean:   mean,
		StdDev: stdDev,
		Max:    max,
		Min:    min,
		Median: median,
	}
}

func ExtractCodeMetrics(code string) CodeMetrics {
	lines := splitLines(code)

	totalChars := len(code)
	lineCount := len(lines)

	if lineCount == 0 {
		return CodeMetrics{}
	}

	avgLineLength := float64(totalChars) / float64(lineCount)

	emptyLines := 0
	for _, line := range lines {
		if len(trimSpace(line)) == 0 {
			emptyLines++
		}
	}
	emptyLineRatio := float64(emptyLines) / float64(lineCount)

	return CodeMetrics{
		LineCount:      lineCount,
		CharCount:      totalChars,
		AvgLineLength:  avgLineLength,
		EmptyLineRatio: emptyLineRatio,
	}
}

func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}

	return s[start:end]
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
