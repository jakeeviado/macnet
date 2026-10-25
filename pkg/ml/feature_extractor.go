// Package ml provides the machine learning pipeline for MACnet, including
// feature extraction, statistical analysis of heuristic signals, and
// interfacing with ONNX-based inference engines.
//
// # INTEGRATION WITH ANALYZER
//
// This package is the bridge between the 'analyzer' package and the ML model.
// The analyzer produces Signal results (heuristics); this package turns them
// into a fixed-length numerical vector (FeatureVector).
//
// Only file-scope signals that actually ran (StatusApplied) contribute to the
// vector. Diff-scope signals use a separate feature set and a separate
// dataset, so they never leak into this one.
//
// # MISSING VALUES
//
// A signal that was skipped, disabled by the active profile, or never ran is
// encoded as Missing (-1), NOT 0. A score of 0 means "ran and found nothing".
// Treating the two the same would teach the model that short or unsupported
// files are clean. Missing signals are excluded from the aggregate statistics.
//
// # MAINTENANCE & SYNCHRONIZATION
//
// When a new file-scope signal is added to the analyzer, it MUST also be
// appended to signalRegistry below. A signal that is not registered is
// ignored by the model. Unregistered reports which signals were dropped, and
// is intended for a unit test or a --verbose warning.
//
// # TRAINING WORKFLOW
//
// The FeatureVector is logged with a ground-truth label (AI versus Human) to
// build a dataset. The model learns to associate score patterns, such as high
// comment density combined with low code complexity, with AI generation.
//
// # FEATURE SCHEME
//
// Features are laid out in a fixed, immutable order to stay compatible with
// the ONNX input tensor:
//
//  1. Language identification (integer mapping)
//  2. Heuristic signal scores (indexed via signalRegistry; Missing if absent)
//  3. Structural code metrics (line count, char count, avg line length,
//     empty line ratio)
//  4. Statistical aggregates over applied signals (mean, stddev, max, min,
//     median) followed by the count of signals with a score above zero
//
// signalRegistry is append-only. Reordering or deleting entries shifts every
// later feature index and invalidates existing models. Appending changes the
// vector length, which also requires retraining.
package ml

import (
	"macnet/pkg/signal"
	"math"
	"slices"
)

// Missing marks a signal that did not contribute to the vector.
// Tree-based models split on it cleanly, and it can never be confused with a
// real score, which is always within [0, 1].
const Missing float32 = -1

// signalRegistry defines the order of signal features in the vector.
// APPEND-ONLY: see the package documentation.
var signalRegistry = [...]string{
	"comment_density", "generic_naming", "repetitive_patterns",
	"code_complexity", "formatting_consistency",
	"comment_redundancy", "emoji_sentiment", "identifier_order", "defensive_ratio",
}

// Vector layout. All offsets derive from the registry size, so the layout
// cannot drift from the registry.
const (
	numSignals = len(signalRegistry)

	offLanguage = 0
	offSignals  = offLanguage + 1
	offMetrics  = offSignals + numSignals // line_count, char_count, avg_line_length, empty_line_ratio
	offStats    = offMetrics + 4          // mean, stddev, max, min, median
	offActive   = offStats + 5            // active_signal_count

	// NumFeatures is the length of every FeatureVector. It must match the
	// input dimension of the ONNX model.
	NumFeatures = offActive + 1
)

// languageIDs maps a language name to its stable integer id.
// Unknown languages fall back to 0 ("unknown").
var languageIDs = map[string]int{
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
}

// FeatureVector is the model input plus its column names.
// Names is shared between vectors; treat it as read-only.
type FeatureVector struct {
	Features []float32
	Names    []string
}

// FeatureExtractor converts analyzer output into FeatureVectors.
// It is immutable after construction and safe for concurrent use.
type FeatureExtractor struct {
	signalIndex map[string]int // signal name -> offset within the signal block
	names       []string       // built once, shared by all vectors
}

// NewFeatureExtractor initializes the extractor with the fixed schema.
//
// WARNING: the order and content of signalRegistry are critical for model
// stability. See the package documentation before editing it.
func NewFeatureExtractor() *FeatureExtractor {
	index := make(map[string]int, numSignals)
	for i, name := range signalRegistry {
		index[name] = i
	}
	return &FeatureExtractor{
		signalIndex: index,
		names:       buildNames(),
	}
}

// Names returns the column names in vector order, e.g. for a CSV header.
// The returned slice is shared; do not modify it.
func (fe *FeatureExtractor) Names() []string { return fe.names }

func buildNames() []string {
	names := make([]string, 0, NumFeatures)
	names = append(names, "language_id")
	names = append(names, signalRegistry[:]...)
	return append(names,
		"line_count", "char_count", "avg_line_length", "empty_line_ratio",
		"signal_mean", "signal_stddev", "signal_max", "signal_min", "signal_median",
		"active_signal_count",
	)
}

// Extract converts signals and code metrics into a FeatureVector.
//
// Only applied, file-scope, registered signals contribute. Any registered
// signal that is absent, skipped, or out of scope is encoded as Missing and
// excluded from the aggregate statistics. If the same signal appears twice,
// the first result wins.
func (fe *FeatureExtractor) Extract(signals []signal.Signal, language string, metrics CodeMetrics) *FeatureVector {
	features := make([]float32, NumFeatures)

	features[offLanguage] = float32(languageIDs[language])

	for i := 0; i < numSignals; i++ {
		features[offSignals+i] = Missing
	}

	scores := make([]float64, 0, numSignals)
	active := 0

	for _, s := range signals {
		if s.Meta.Scope != signal.ScopeFile || s.Status != signal.StatusApplied {
			continue
		}
		i, ok := fe.signalIndex[s.Meta.Name]
		if !ok || features[offSignals+i] != Missing {
			continue
		}
		features[offSignals+i] = float32(s.Score)
		scores = append(scores, s.Score)
		if s.Score > 0 {
			active++
		}
	}

	features[offMetrics+0] = float32(metrics.LineCount)
	features[offMetrics+1] = float32(metrics.CharCount)
	features[offMetrics+2] = float32(metrics.AvgLineLength)
	features[offMetrics+3] = float32(metrics.EmptyLineRatio)

	// With no applied signals every aggregate is 0, matching the old behavior.
	stats := calculateStats(scores)
	features[offStats+0] = float32(stats.Mean)
	features[offStats+1] = float32(stats.StdDev)
	features[offStats+2] = float32(stats.Max)
	features[offStats+3] = float32(stats.Min)
	features[offStats+4] = float32(stats.Median)

	features[offActive] = float32(active)

	return &FeatureVector{Features: features, Names: fe.names}
}

// Unregistered returns the names of applied file-scope signals that are not
// in signalRegistry, i.e. signals the model is silently ignoring. Use it in a
// test that runs every analyzer check, or as a --verbose warning.
func (fe *FeatureExtractor) Unregistered(signals []signal.Signal) []string {
	var missing []string
	for _, s := range signals {
		if s.Meta.Scope != signal.ScopeFile || s.Status != signal.StatusApplied {
			continue
		}
		if _, ok := fe.signalIndex[s.Meta.Name]; !ok && !slices.Contains(missing, s.Meta.Name) {
			missing = append(missing, s.Meta.Name)
		}
	}
	return missing
}

// CodeMetrics are cheap structural measurements of a source file.
type CodeMetrics struct {
	LineCount      int
	CharCount      int
	AvgLineLength  float64
	EmptyLineRatio float64
}

// Stats are aggregate statistics over a set of signal scores.
type Stats struct {
	Mean   float64
	StdDev float64
	Max    float64
	Min    float64
	Median float64
}

// calculateStats computes population statistics. An empty input yields zeros.
func calculateStats(scores []float64) Stats {
	n := len(scores)
	if n == 0 {
		return Stats{}
	}

	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	mean := sum / float64(n)

	variance := 0.0
	for _, s := range scores {
		d := s - mean
		variance += d * d
	}
	variance /= float64(n)

	sorted := slices.Clone(scores)
	slices.Sort(sorted)

	var median float64
	if n%2 == 0 {
		median = (sorted[n/2-1] + sorted[n/2]) / 2.0
	} else {
		median = sorted[n/2]
	}

	return Stats{
		Mean:   mean,
		StdDev: math.Sqrt(variance),
		Max:    sorted[n-1],
		Min:    sorted[0],
		Median: median,
	}
}

// ExtractCodeMetrics measures line and character statistics for a file.
//
// The line splitting and whitespace rules are intentionally the original
// ones, NOT strings.Split or unicode-aware trimming: changing them shifts
// line_count and empty_line_ratio for every file and silently moves the
// feature distribution the current model was trained on.
func ExtractCodeMetrics(code string) CodeMetrics {
	lines := splitLines(code)

	lineCount := len(lines)
	if lineCount == 0 {
		return CodeMetrics{}
	}

	totalChars := len(code)

	emptyLines := 0
	for _, line := range lines {
		if len(trimSpace(line)) == 0 {
			emptyLines++
		}
	}

	return CodeMetrics{
		LineCount:      lineCount,
		CharCount:      totalChars,
		AvgLineLength:  float64(totalChars) / float64(lineCount),
		EmptyLineRatio: float64(emptyLines) / float64(lineCount),
	}
}

// splitLines splits on '\n' and drops a trailing empty segment, unlike
// strings.Split, which would add one.
func splitLines(s string) []string {
	lines := make([]string, 0, 64)
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

// trimSpace trims ASCII whitespace only.
func trimSpace(s string) string {
	start, end := 0, len(s)
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
