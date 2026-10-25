package signals

import (
	"macnet/pkg/signal"
	"regexp"
	"strings"
)

var genericNamingMeta = signal.SignalMeta{
	ID:          "F2",
	Name:        "generic_naming",
	Description: "Placeholder names",
	Scope:       signal.ScopeFile,
	Class:       signal.ClassLexical,
	Tier:        signal.TierFile,
	MinProfile:  signal.ProfileLoose,
	Weight:      1.0,
}

var genericNameRegex = regexp.MustCompile(
	`(?i)\b(?:temp|tmp|data|result|item|value|obj|elem|variable|parameter|foo|bar|test|example)\b`,
)

// CheckGenericNaming flags heavy use of placeholder names.
func CheckGenericNaming(code string) signal.Signal {
	if strings.TrimSpace(code) == "" {
		return signal.Skipped(genericNamingMeta, "empty file")
	}

	matches := len(genericNameRegex.FindAllStringIndex(code, -1))
	lines := strings.Count(code, "\n") + 1
	density := float64(matches) / float64(lines)

	var score float64
	var detail string

	switch {
	case density > 0.5:
		score, detail = 0.9, "Very high use of generic variable names"
	case density > 0.2:
		score, detail = 0.6, "High use of generic variable names"
	default:
		return signal.NewSignal(genericNamingMeta, 0.0)
	}

	return signal.NewSignal(genericNamingMeta, score, signal.Evidence{
		Detail: detail,
		Metrics: map[string]float64{
			"generic_count":   float64(matches),
			"generic_density": density,
		},
	})
}
