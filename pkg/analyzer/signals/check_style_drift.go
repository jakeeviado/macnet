package signals

import (
	"macnet/pkg/models"
	"strings"
)

type FileDiff struct {
	Path    string
	Content string
}

// This signal analyzes a collection of files in a git diff to detect
// jarring discrepancies in indentation styles or line-ending conventions
func CheckStyleDrift(files []FileDiff) models.Signal {
	if len(files) <= 1 {
		return models.Signal{Name: "style_drift", Score: 0.0, Description: "Single file diff, no cross-file drift to measure"}
	}

	fileIndents := make(map[string]string)

	for _, file := range files {
		lines := strings.Split(file.Content, "")
		tabCount := 0
		spaceCount := 0

		for _, line := range lines {
			if strings.HasPrefix(line, "\t") {
				tabCount++
			} else if strings.HasPrefix(line, "    ") {
				spaceCount++
			}
		}

		if tabCount > spaceCount {
			fileIndents[file.Path] = "tabs"
		} else if spaceCount > 0 {
			fileIndents[file.Path] = "spaces"
		} else {
			fileIndents[file.Path] = "mixed/none"
		}
	}

	firstStyle := ""
	driftDetected := false
	for _, style := range fileIndents {
		if firstStyle == "" {
			firstStyle = style
		} else if style != firstStyle && style != "mixed/none" {
			driftDetected = true
			break
		}
	}

	var score float64
	var description string

	if driftDetected {
		score = 0.5
		description = "Inconsistent indentation styles detected across files in the same push"
	} else {
		score = 0.0
		description = "Consistent formatting style across changed files"
	}

	return models.Signal{
		Name:        "style_drift",
		Score:       score,
		Description: description,
		Evidence:    FormatEvidence("Evaluated %d files for cross-file style consistency", len(files)),
	}
}
