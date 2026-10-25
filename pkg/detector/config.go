package detector

import "macnet/pkg/signal"

// DetectorConfiguration holds settings for scan execution, filtering, and optional model integration.
type DetectorConfiguration struct {
	Threshold    float64
	Languages    []string
	ExcludePaths []string
	IsVerbose    bool
	ModelPath    string
	UseML        bool
	MLWeight     float64
	IsMLOnly     bool
}

// FileResult contains analysis metrics, heuristic signals, scores, and execution metadata for a single scanned file.
type FileResult struct {
	Path           string
	Score          float64
	Signals        []signal.Signal
	Language       string
	Error          error
	MLScore        float64
	HeuristicScore float64
	UsedML         bool
	Confidence     float64
}

// ScanResults aggregates individual file results, summary counts, and batch statistics for an entire scan operation.
type ScanResults struct {
	Files         []FileResult
	TotalFiles    int
	DetectedFiles int
	AverageScore  float64
	UsedML        bool
}

// HasDetections checks whether any scanned file in the report met or exceeded the specified risk threshold.
func (r *ScanResults) HasDetections(threshold float64) bool {
	for _, file := range r.Files {
		if file.Score >= threshold {
			return true
		}
	}
	return false
}
