package signals

import (
	"macnet/pkg/models"
	"regexp"
	"strings"
)

var (
	identifierRegex = regexp.MustCompile(`^[A-Z_a-z0-9]+\s*[:=]`)
	assignmentRegex = regexp.MustCompile(`[:=]`)
)

// CheckIdentifierOrder checks whether large blocks of identifiers are perfectly alphabetically sorted.
func CheckIdentifierOrder(code string) models.Signal {
	lines := strings.Split(code, "\n")
	var currentBlock []string
	perfectlySortedBlocks := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if identifierRegex.MatchString(trimmed) {
			parts := assignmentRegex.Split(trimmed, 2)
			name := strings.TrimSpace(parts[0])
			currentBlock = append(currentBlock, name)
		} else {
			if len(currentBlock) >= 6 {
				if isSorted(currentBlock) {
					perfectlySortedBlocks++
				}
			}
			currentBlock = []string{}
		}
	}

	var score float64
	if perfectlySortedBlocks > 0 {
		score = 0.6
	}

	return models.Signal{
		Name:        "identifier_order",
		Score:       score,
		Description: "Large identifier blocks in perfect alphabetical order warrant a closer look",
		Evidence:    FormatEvidence("Perfectly sorted blocks (6+ items): %d", perfectlySortedBlocks),
	}
}

func isSorted(list []string) bool {
	for i := 1; i < len(list); i++ {
		if list[i] < list[i-1] {
			return false
		}
	}
	return true
}
