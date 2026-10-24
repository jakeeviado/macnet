package signals

import (
	"macnet/pkg/models"
)

type DiffStats struct {
	Additions    int
	Deletions    int
	FilesChanged int
}

// This signal evaluates the balance of code added vs. code removed.
// Unnaturally high addition-to-deletion ratios can indicate un-refactored, generated code dumps.
func CheckAdditionDeletionRatio(stats DiffStats) models.Signal {
	if stats.Additions == 0 {
		return models.Signal{
			Name:        "addition_deletion_ratio",
			Score:       0.0,
			Description: "No additions in diff",
		}
	}

	deletionRatio := float64(stats.Deletions) / float64(stats.Additions)

	var score float64
	var description string

	if stats.Additions > 150 && deletionRatio < 0.02 {
		score = 0.7
		description = "Massive code addition with near-zero refactoring or deletion"
	} else if stats.Additions > 300 && deletionRatio < 0.05 {
		score = 0.4
		description = "Very high net-positive code addition relative to deletions"
	} else {
		score = 0.0
		description = "Healthy balance of code additions and deletions"
	}

	return models.Signal{
		Name:        "addition_deletion_ratio",
		Score:       score,
		Description: description,
		Evidence:    FormatEvidence("Added: +%d, Deleted: -%d (Deletion ratio: %.2f)", stats.Additions, stats.Deletions, deletionRatio),
	}
}
