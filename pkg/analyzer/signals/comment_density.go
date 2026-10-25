package signals

import (
	"macnet/pkg/signal"
	"strings"
)

var commentDensityMeta = signal.SignalMeta{
	ID:          "F1",
	Name:        "comment_density",
	Description: "Comment-to-code ratio",
	Scope:       signal.ScopeFile,
	Class:       signal.ClassStylistic,
	Tier:        signal.TierFile,
	MinProfile:  signal.ProfileLoose,
	Weight:      1.0,
}

// CheckCommentDensity calculates the comment-to-code ratio. Unusually high
// density can be a risk signal in review pipelines.
func CheckCommentDensity(code string) signal.Signal {
	if strings.TrimSpace(code) == "" {
		return signal.Skipped(commentDensityMeta, "empty file")
	}

	lines := strings.Split(code, "\n")
	commentLines := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") ||
			strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "*") {
			commentLines++
		}
	}

	density := float64(commentLines) / float64(len(lines))

	var score float64
	var detail string

	switch {
	case density > 0.3:
		score, detail = 0.8, "Excessive comment density detected (>30%)"
	case density > 0.2:
		score, detail = 0.5, "High comment density (>20%)"
	default:
		return signal.NewSignal(commentDensityMeta, 0.0)
	}

	return signal.NewSignal(commentDensityMeta, score, signal.Evidence{
		Detail: detail,
		Metrics: map[string]float64{
			"comment_lines":   float64(commentLines),
			"total_lines":     float64(len(lines)),
			"comment_density": density,
		},
	})
}
