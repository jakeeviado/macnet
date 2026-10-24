package signals

import (
	"macnet/pkg/models"
	"strings"
)

// Evaluates indentation uniformity. Suspiciously rigid
// consistency can indicate non-human authorship.
func CheckFormattingConsistency(code string) models.Signal {
	lines := strings.Split(code, "\n")
	if len(lines) < 5 {
		return models.Signal{Name: "formatting_consistency", Score: 0.0}
	}

	indentations := make(map[int]int)
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		indent := 0
		for _, char := range line {
			if char == ' ' || char == '\t' {
				indent++
			} else {
				break
			}
		}
		indentations[indent]++
	}

	uniqueIndents := len(indentations)

	var score float64
	var description string

	if uniqueIndents <= 3 && len(lines) > 20 {
		score = 0.5
		description = "Unusually rigid indentation uniformity across a large file"
	} else {
		score = 0.0
		description = "Normal formatting variation"
	}

	return models.Signal{
		Name:        "formatting_consistency",
		Score:       score,
		Description: description,
		Evidence:    FormatEvidence("Unique indentation levels: %d", uniqueIndents),
	}
}
