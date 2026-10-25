package signals

import (
	"macnet/pkg/signal"
	"strings"
)

var styleDriftMeta = signal.SignalMeta{
	ID:          "G9",
	Name:        "style_drift",
	Description: "Inconsistent indentation style across files in one diff",
	Scope:       signal.ScopeDiff,
	Class:       signal.ClassStylistic,
	Tier:        signal.TierDiff,
	MinProfile:  signal.ProfileStrict,
	Advisory:    true,
	Weight:      1.0,
}

type FileDiff struct {
	Path    string
	Content string
}

// CheckStyleDrift analyzes the files in a diff for jarring discrepancies
// in indentation style (tabs vs spaces) between files.
func CheckStyleDrift(files []FileDiff) signal.Signal {
	if len(files) <= 1 {
		return signal.Skipped(styleDriftMeta, "single-file diff, no cross-file drift to measure")
	}

	var tabFiles, spaceFiles []string

	for _, file := range files {
		tabCount, spaceCount := 0, 0
		for _, line := range strings.Split(file.Content, "\n") {
			if strings.HasPrefix(line, "\t") {
				tabCount++
			} else if strings.HasPrefix(line, "    ") {
				spaceCount++
			}
		}

		switch {
		case tabCount > spaceCount:
			tabFiles = append(tabFiles, file.Path)
		case spaceCount > 0:
			spaceFiles = append(spaceFiles, file.Path)
		}
	}

	if len(tabFiles) == 0 || len(spaceFiles) == 0 {
		return signal.NewSignal(styleDriftMeta, 0.0)
	}

	outliers, label := tabFiles, "indents with tabs"
	if len(spaceFiles) < len(tabFiles) {
		outliers, label = spaceFiles, "indents with spaces"
	}

	evidence := make([]signal.Evidence, 0, len(outliers))
	for _, p := range outliers {
		evidence = append(evidence, signal.Evidence{
			Path:   p,
			Detail: label + " while other changed files do not",
			Metrics: map[string]float64{
				"tab_files":   float64(len(tabFiles)),
				"space_files": float64(len(spaceFiles)),
			},
		})
	}

	return signal.NewSignal(styleDriftMeta, 0.5, evidence...)
}
