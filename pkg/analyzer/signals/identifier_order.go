package signals

import (
	"macnet/pkg/signal"
	"regexp"
	"strings"
)

var identifierOrderMeta = signal.SignalMeta{
	ID:          "F8",
	Name:        "identifier_order",
	Description: "Suspiciously perfect alphabetical ordering",
	Scope:       signal.ScopeFile,
	Class:       signal.ClassLexical,
	Tier:        signal.TierFile,
	MinProfile:  signal.ProfileLoose,
	Weight:      1.0,
}

var (
	identifierRegex = regexp.MustCompile(`^[A-Z_a-z0-9]+\s*[:=]`)
	assignmentRegex = regexp.MustCompile(`[:=]`)
)

// CheckIdentifierOrder checks whether large blocks of identifiers are perfectly alphabetically sorted.
func CheckIdentifierOrder(code string) signal.Signal {
	lines := strings.Split(code, "\n")

	var block []string
	var blockStart int
	var evidence []signal.Evidence

	flush := func() {
		if len(block) >= 6 && isSorted(block) {
			evidence = append(evidence, signal.Evidence{
				Line:   blockStart,
				Detail: "Perfectly sorted identifier block",
				Metrics: map[string]float64{
					"block_size": float64(len(block)),
				},
			})
		}
		block = block[:0]
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if identifierRegex.MatchString(trimmed) {
			if len(block) == 0 {
				blockStart = i + 1
			}
			parts := assignmentRegex.Split(trimmed, 2)
			block = append(block, strings.TrimSpace(parts[0]))
		} else {
			flush()
		}
	}
	flush()

	if len(evidence) == 0 {
		return signal.NewSignal(identifierOrderMeta, 0.0)
	}
	return signal.NewSignal(identifierOrderMeta, 0.6, evidence...)
}

func isSorted(list []string) bool {
	for i := 1; i < len(list); i++ {
		if strings.ToLower(list[i]) < strings.ToLower(list[i-1]) {
			return false
		}
	}
	return true
}
