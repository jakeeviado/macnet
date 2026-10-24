package signals

import (
	"macnet/pkg/models"
	"strings"
)

// Flags inline comments that closely mirror the surrounding code.
// High overlap can indicate over-explained or low-signal documentation.
func CheckCommentRedundancy(code string) models.Signal {
	lines := strings.Split(code, "\n")
	redundantCount := 0
	totalComments := 0

	for _, line := range lines {
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

		if commentPart != "" {
			totalComments++
			if codePart != "" {
				cCode := strings.ToLower(codePart)
				cComment := strings.ToLower(commentPart)
				words := strings.Fields(cComment)
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
		}
	}

	var score float64
	if totalComments > 0 {
		score = float64(redundantCount) / float64(totalComments)
	}

	return models.Signal{
		Name:        "comment_redundancy",
		Score:       score,
		Description: "High comment-to-code overlap may indicate low-quality or over-explained code",
		Evidence:    FormatEvidence("Redundant comments: %d / %d", redundantCount, totalComments),
	}
}
