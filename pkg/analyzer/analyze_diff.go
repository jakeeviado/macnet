package analyzer

import (
	"macnet/pkg/analyzer/signals"
	"macnet/pkg/signal"
)

// AnalyzeDiff runs the diff-scope checks that are active in the analyzer's
// profile across the whole git diff context.
func (a *Analyzer) AnalyzeDiff(files []signals.FileDiff, stats signals.DiffStats) []signal.Signal {
	ctx := signals.DiffContext{Files: files, Stats: stats}

	sigList := make([]signal.Signal, 0, len(signals.DiffChecks))
	for _, c := range signals.DiffChecks {
		if !c.Meta.ActiveIn(a.profile) {
			continue
		}
		sigList = append(sigList, c.Run(ctx))
	}
	return sigList
}
