package signals

import (
	"macnet/pkg/signal"
	"strings"
)

const minLinesForFormatting = 20

var formattingConsistencyMeta = signal.SignalMeta{
	ID:          "F5",
	Name:        "formatting_consistency",
	Description: "Indentation and style uniformity",
	Scope:       signal.ScopeFile,
	Class:       signal.ClassStylistic,
	Tier:        signal.TierFile,
	MinProfile:  signal.ProfileLoose,
	Weight:      1.0,
}

// CheckFormattingConsistency evaluates indentation uniformity.
func CheckFormattingConsistency(code string) signal.Signal {
	indentations := make(map[int]int)
	nonBlank := 0

	for _, line := range strings.Split(code, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonBlank++

		indent := 0
		for _, char := range line {
			if char != ' ' && char != '\t' {
				break
			}
			indent++
		}
		indentations[indent]++
	}

	if nonBlank < minLinesForFormatting {
		return signal.Skipped(formattingConsistencyMeta, "file too short to judge indentation")
	}

	uniqueIndents := len(indentations)
	if uniqueIndents > 3 {
		return signal.NewSignal(formattingConsistencyMeta, 0.0)
	}

	return signal.NewSignal(formattingConsistencyMeta, 0.5, signal.Evidence{
		Detail: "Unusually rigid indentation uniformity across a large file",
		Metrics: map[string]float64{
			"unique_indent_levels": float64(uniqueIndents),
			"non_blank_lines":      float64(nonBlank),
		},
	})
}
