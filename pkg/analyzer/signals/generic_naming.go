package signals

import (
	"macnet/pkg/models"
	"regexp"
	"strings"
)

// CheckGenericNaming flags heavy use of placeholder names.
func CheckGenericNaming(code string) models.Signal {
	genericNames := []string{
		"temp", "tmp", "data", "result", "item", "value", "obj", "elem",
		"variable", "parameter", "foo", "bar", "test", "example",
	}

	matches := 0
	for _, name := range genericNames {
		pattern := `\b` + name + `\b`
		re := regexp.MustCompile(pattern)
		matches += len(re.FindAllString(strings.ToLower(code), -1))
	}

	lines := len(strings.Split(code, "\n"))
	if lines == 0 {
		return models.Signal{Name: "generic_naming", Score: 0.0}
	}

	density := float64(matches) / float64(lines)

	var score float64
	var description string

	if density > 0.5 {
		score = 0.9
		description = "Very high use of generic variable names"
	} else if density > 0.2 {
		score = 0.6
		description = "High use of generic variable names"
	} else {
		score = 0.0
		description = "Normal variable naming patterns"
	}

	return models.Signal{
		Name:        "generic_naming",
		Score:       score,
		Description: description,
		Evidence:    FormatEvidence("Generic name occurrences: %d", matches),
	}
}
