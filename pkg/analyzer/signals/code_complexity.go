package signals

import (
	"sentinel/pkg/models"
	"strings"
)

// Measures control flow density across the file.
// Unusually low complexity can indicate shallow or auto-generated logic.
func CheckCodeComplexity(code string) models.Signal {
	controlFlowKeywords := []string{
		"if", "else", "switch", "case", "for", "while", "do",
		"try", "catch", "finally", "throw",
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

	if len(lines) == 0 {
		return models.Signal{Name: "code_complexity", Score: 0.0}
	}

	complexity := float64(controlFlowCount) / float64(len(lines))

	var score float64
	var description string

	if complexity < 0.05 {
		score = 0.6
		description = "Unusually low cyclomatic complexity"
	} else if complexity < 0.1 {
		score = 0.3
		description = "Low cyclomatic complexity"
	} else {
		score = 0.0
		description = "Normal code complexity"
	}

	return models.Signal{
		Name:        "code_complexity",
		Score:       score,
		Description: description,
		Evidence:    FormatEvidence("Control flow density: %.2f", complexity),
	}
}
