package analyzer

import (
	"macnet/pkg/analyzer/signals"
	"macnet/pkg/models"
)

// AnalyzeDiff runs macro-level checks across the entire git diff context.
func (a *Analyzer) AnalyzeDiff(files []signals.FileDiff, stats signals.DiffStats) []models.Signal {
	sigList := make([]models.Signal, 0)
	sigList = append(sigList, signals.CheckAdditionDeletionRatio(stats))
	sigList = append(sigList, signals.CheckStyleDrift(files))
	return sigList
}
