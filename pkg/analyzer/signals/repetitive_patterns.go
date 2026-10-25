package signals

import (
	"macnet/pkg/signal"
	"strings"
)

var repetitivePatternsMeta = signal.SignalMeta{
	ID:          "F3",
	Name:        "repetitive_patterns",
	Description: "File-level code duplication",
	Scope:       signal.ScopeFile,
	Class:       signal.ClassStructural,
	Tier:        signal.TierFile,
	MinProfile:  signal.ProfileLoose,
	Weight:      1.0,
}

// Looks for duplicated lines across the file.
// High repetition often signals copy-paste patterns or low code quality.
func CheckRepetitivePatterns(code string) signal.Signal {
	lines := strings.Split(code, "\n")
	if len(lines) < 10 {
		return signal.Skipped(repetitivePatternsMeta, "file under 10 lines")
	}

	lineCount := make(map[string]int)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 10 {
			lineCount[trimmed]++
		}
	}

	repetitions := 0
	for _, count := range lineCount {
		if count > 2 {
			repetitions += count - 1
		}
	}

	ratio := float64(repetitions) / float64(len(lines))

	var score float64
	var detail string

	switch {
	case ratio > 0.3:
		score, detail = 0.7, "High code repetition detected"
	case ratio > 0.15:
		score, detail = 0.4, "Moderate code repetition"
	default:
		return signal.NewSignal(repetitivePatternsMeta, 0.0)
	}

	return signal.NewSignal(repetitivePatternsMeta, score, signal.Evidence{
		Detail: detail,
		Metrics: map[string]float64{
			"repetition_ratio": ratio,
			"repeated_lines":   float64(repetitions),
		},
	})
}
