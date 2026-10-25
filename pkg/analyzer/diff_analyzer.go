package analyzer

import (
	"macnet/pkg/analyzer/signals"
	"macnet/pkg/signal"
)

// AnalyzeDiff runs macro-level checks across the entire git diff context.
func (a *Analyzer) AnalyzeDiff(files []signals.FileDiff, stats signals.DiffStats) []signal.Signal {
	sigList := make([]signal.Signal, 0)
	sigList = append(sigList, signals.CheckAdditionDeletionRatio(stats))
	sigList = append(sigList, signals.CheckStyleDrift(files))
	return sigList
}
