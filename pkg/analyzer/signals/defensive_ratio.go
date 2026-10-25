package signals

import (
	"macnet/pkg/signal"
	"strings"
)

var defensiveRatioMeta = signal.SignalMeta{
	ID:          "F9",
	Name:        "defensive_ratio",
	Description: "Ratio of error handling to functional logic",
	Scope:       signal.ScopeFile,
	Class:       signal.ClassStructural,
	Tier:        signal.TierFile,
	MinProfile:  signal.ProfileLoose,
	Weight:      1.0,
}

var defensePatterns = []string{
	"if err != nil", "if (err", "if (!err", "if (obj == null)",
	"if (typeof", "try {", "catch (", "if not ", "if (!",
}

// CheckDefensiveRatio evaluates the ratio of defensive checks to functional logic.
// A heavily skewed ratio can indicate over-cautious or templated error handling.
func CheckDefensiveRatio(code string) signal.Signal {
	defenseCount, logicCount := 0, 0

	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}

		isDefense := false
		for _, p := range defensePatterns {
			if strings.Contains(trimmed, p) {
				isDefense = true
				break
			}
		}

		if isDefense {
			defenseCount++
		} else if strings.Contains(trimmed, "=") || strings.Contains(trimmed, "(") {
			logicCount++
		}
	}

	if logicCount == 0 {
		return signal.Skipped(defensiveRatioMeta, "no functional logic lines found")
	}

	ratio := float64(defenseCount) / float64(logicCount)

	var score float64
	var detail string

	switch {
	case ratio > 0.5 && logicCount > 5:
		score, detail = 0.7, "Disproportionately high ratio of error checks to functional logic"
	case ratio > 0.3:
		score, detail = 0.4, "Elevated defensive check density relative to logic"
	default:
		return signal.NewSignal(defensiveRatioMeta, 0.0)
	}

	return signal.NewSignal(defensiveRatioMeta, score, signal.Evidence{
		Detail: detail,
		Metrics: map[string]float64{
			"defense_lines": float64(defenseCount),
			"logic_lines":   float64(logicCount),
			"defense_ratio": ratio,
		},
	})
}
