package signals

import (
	"macnet/pkg/models"
	"strings"
)

// Evaluates the ratio of defensive checks to functional logic.
// A heavily skewed ratio can indicate over-cautious or templated error handling.
func CheckDefensiveRatio(code string) models.Signal {
	lines := strings.Split(code, "\n")
	defensePatterns := []string{
		"if err != nil", "if (err", "if (!err", "if (obj == null)",
		"if (typeof", "try {", "catch (", "if not ", "if (!",
	}

	defenseCount := 0
	logicCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}

		isDefense := false
		for _, pattern := range defensePatterns {
			if strings.Contains(trimmed, pattern) {
				defenseCount++
				isDefense = true
				break
			}
		}

		if !isDefense && (strings.Contains(trimmed, "=") || strings.Contains(trimmed, "(")) {
			logicCount++
		}
	}

	ratio := 0.0
	if logicCount > 0 {
		ratio = float64(defenseCount) / float64(logicCount)
	}

	var score float64
	var description string

	if ratio > 0.5 && logicCount > 5 {
		score = 0.7
		description = "Disproportionately high ratio of error checks to functional logic"
	} else if ratio > 0.3 {
		score = 0.4
		description = "Elevated defensive check density relative to logic"
	}

	return models.Signal{
		Name:        "defensive_ratio",
		Score:       score,
		Description: description,
		Evidence:    FormatEvidence("Safety checks: %d, Logic lines: %d (Ratio: %.2f)", defenseCount, logicCount, ratio),
	}
}
