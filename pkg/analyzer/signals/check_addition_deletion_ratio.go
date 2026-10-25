package signals

import (
	"macnet/pkg/signal"
)

var additionDeletionMeta = signal.SignalMeta{
	ID:          "G14",
	Name:        "addition_deletion_ratio",
	Description: "Large additions with little deletion or revision",
	Scope:       signal.ScopeDiff,
	Class:       signal.ClassProcess,
	Tier:        signal.TierDiff,
	MinProfile:  signal.ProfileStrict,
	Advisory:    true,
	Weight:      1.0,
}

type DiffStats struct {
	Additions    int
	Deletions    int
	FilesChanged int
}

// CheckAdditionDeletionRatio evaluates the balance of code added vs. removed.
// Unnaturally high addition-to-deletion ratios can indicate un-refactored,
// generated code dumps.
func CheckAdditionDeletionRatio(stats DiffStats) signal.Signal {
	if stats.Additions == 0 {
		return signal.Skipped(additionDeletionMeta, "no additions in diff")
	}

	deletionRatio := float64(stats.Deletions) / float64(stats.Additions)

	var score float64
	var detail string

	switch {
	case stats.Additions > 150 && deletionRatio < 0.02:
		score, detail = 0.7, "Massive code addition with near-zero refactoring or deletion"
	case stats.Additions > 300 && deletionRatio < 0.05:
		score, detail = 0.4, "Very high net-positive code addition relative to deletions"
	default:
		return signal.NewSignal(additionDeletionMeta, 0.0)
	}

	return signal.NewSignal(additionDeletionMeta, score, signal.Evidence{
		Detail: detail,
		Metrics: map[string]float64{
			"additions":      float64(stats.Additions),
			"deletions":      float64(stats.Deletions),
			"deletion_ratio": deletionRatio,
			"files_changed":  float64(stats.FilesChanged),
		},
	})
}
