package signals

import (
	"macnet/pkg/signal"
	"strings"
)

var commentRedundancyMeta = signal.SignalMeta{
	ID:          "F6",
	Name:        "comment_redundancy",
	Description: "Comments that mirror code logic",
	Scope:       signal.ScopeFile,
	Class:       signal.ClassStylistic,
	Tier:        signal.TierFile,
	MinProfile:  signal.ProfileLoose,
	Weight:      1.0,
}

// CheckCommentRedundancy flags inline comments that closely mirror the surrounding code.
// High overlap can indicate over-explained or low-signal documentation.
func CheckCommentRedundancy(code string) signal.Signal {
	redundantCount, totalComments := 0, 0

	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var codePart, commentPart string
		if idx := strings.Index(line, "//"); idx != -1 {
			codePart = strings.TrimSpace(line[:idx])
			commentPart = strings.TrimSpace(line[idx+2:])
		} else if idx := strings.Index(line, "#"); idx != -1 {
			codePart = strings.TrimSpace(line[:idx])
			commentPart = strings.TrimSpace(line[idx+1:])
		}

		if commentPart == "" {
			continue
		}
		totalComments++
		if codePart == "" {
			continue
		}

		cCode := strings.ToLower(codePart)
		words := strings.Fields(strings.ToLower(commentPart))
		matchCount := 0
		for _, word := range words {
			if len(word) > 2 && strings.Contains(cCode, word) {
				matchCount++
			}
		}

		if len(words) > 0 && float64(matchCount)/float64(len(words)) > 0.5 {
			redundantCount++
		}
	}

	if totalComments == 0 {
		return signal.Skipped(commentRedundancyMeta, "no comments found")
	}
	if redundantCount == 0 {
		return signal.NewSignal(commentRedundancyMeta, 0.0)
	}

	return signal.NewSignal(commentRedundancyMeta,
		float64(redundantCount)/float64(totalComments),
		signal.Evidence{
			Detail: "High comment-to-code overlap may indicate over-explained code",
			Metrics: map[string]float64{
				"redundant_comments": float64(redundantCount),
				"total_comments":     float64(totalComments),
			},
		})
}
