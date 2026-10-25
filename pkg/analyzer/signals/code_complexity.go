package signals

import (
	"macnet/pkg/signal"
	"strings"
)

var codeComplexityMeta = signal.SignalMeta{
	ID:          "F4",
	Name:        "code_complexity",
	Description: "Control flow density",
	Scope:       signal.ScopeFile,
	Class:       signal.ClassStructural,
	Tier:        signal.TierFile,
	MinProfile:  signal.ProfileLoose,
	Weight:      1.0,
}

var controlFlowKeywords = []string{
	"if", "else", "switch", "case", "for", "while", "do",
	"try", "catch", "finally", "throw",
}

// CheckCodeComplexity measures control flow density across the file.
// Unusually low complexity can indicate shallow or auto-generated logic.
func CheckCodeComplexity(code string) signal.Signal {
	if strings.TrimSpace(code) == "" {
		return signal.Skipped(codeComplexityMeta, "empty file")
	}

	lines := strings.Split(code, "\n")
	controlFlowCount := 0

	for _, line := range lines {
		lower := strings.ToLower(line)
		for _, keyword := range controlFlowKeywords {
			if strings.Contains(lower, keyword) {
				controlFlowCount++
				break
			}
		}
	}

	complexity := float64(controlFlowCount) / float64(len(lines))

	var score float64
	var detail string

	switch {
	case complexity < 0.05:
		score, detail = 0.6, "Unusually low control flow density"
	case complexity < 0.1:
		score, detail = 0.3, "Low control flow density"
	default:
		return signal.NewSignal(codeComplexityMeta, 0.0)
	}

	return signal.NewSignal(codeComplexityMeta, score, signal.Evidence{
		Detail: detail,
		Metrics: map[string]float64{
			"control_flow_lines":   float64(controlFlowCount),
			"control_flow_density": complexity,
		},
	})
}
