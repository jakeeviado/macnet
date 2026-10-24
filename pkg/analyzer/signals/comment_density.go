package signals

import (
	"macnet/pkg/models"
	"strings"
)

// Calculates comment-to-code ratio. Unusually high density
// can be a risk signal in review pipelines.
func CheckCommentDensity(code string) models.Signal {
	lines := strings.Split(code, "\n")
	if len(lines) == 0 {
		return models.Signal{Name: "comment_density", Score: 0.0}
	}

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
	var description string

	if density > 0.3 {
		score = 0.8
		description = "Excessive comment density detected (>30%)"
	} else if density > 0.2 {
		score = 0.5
		description = "High comment density (>20%)"
	} else {
		score = 0.0
		description = "Normal comment density"
	}

	return models.Signal{
		Name:        "comment_density",
		Score:       score,
		Description: description,
		Evidence:    FormatEvidence("Comment lines: %d / %d (%.1f%%)", commentLines, len(lines), density*100),
	}
}
