package signals

import (
	"sentinel/pkg/models"
	"strings"
)

// Looks for duplicated lines across the file.
// High repetition often signals copy-paste patterns or low code quality.
func CheckRepetitivePatterns(code string) models.Signal {
	lines := strings.Split(code, "\n")
	if len(lines) < 10 {
		return models.Signal{Name: "repetitive_patterns", Score: 0.0}
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

	repetitionRatio := float64(repetitions) / float64(len(lines))

	var score float64
	var description string

	if repetitionRatio > 0.3 {
		score = 0.7
		description = "High code repetition detected"
	} else if repetitionRatio > 0.15 {
		score = 0.4
		description = "Moderate code repetition"
	} else {
		score = 0.0
		description = "Normal code variation"
	}

	return models.Signal{
		Name:        "repetitive_patterns",
		Score:       score,
		Description: description,
		Evidence:    FormatEvidence("Repetition ratio: %.1f%%", repetitionRatio*100),
	}
}
