package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"macnet/pkg/detector"
	"macnet/pkg/signal"
)

const (
	lineWidth = 80

	// reviewFraction is the share of the threshold above which a file is
	// listed under "review recommended".
	reviewFraction = 0.6

	// maxEvidencePerSignal caps how many evidence entries are printed per
	// signal in text mode, so a diff signal that lists many files stays readable.
	maxEvidencePerSignal = 5

	// jsonSchemaVersion is bumped whenever the JSON output shape changes.
	// v2: signal evidence is a list of objects instead of a string, and
	// signals carry id, class, scope, advisory, and status.
	jsonSchemaVersion = 2
)

var (
	heavyRule = strings.Repeat("=", lineWidth)
	lightRule = strings.Repeat("-", lineWidth)
)

var logoLines = []string{
	"░▄█ ▀▀▄ ▀▀█  ▀▀▄ ▀▀█  ▀▀▀",
	"█ ░ ░ ▓ ░    █ ▓ ░     ░",
	"▓ ▒ ▒▀▒ ▒    ▓ ▒ ▒▀    ▒",
	"▒ ▓ ▓ ░ ▓ ▄  ▒ ░ ▓ ▄   ▓",
	"▀ ▀ ▀ ▀ ▀▀▀  ░ ▀ ▀▀█   ░",
	"        ▀▀▀   ▀▀▀ ▀▀▀",
	" MACnet Analysis Report",
}

type Config struct {
	JSONOutput bool
	Verbose    bool
	Threshold  float64
}

type Reporter struct {
	config Config
	out    io.Writer
}

// New returns a Reporter that writes to stdout.
func New(config Config) *Reporter {
	return NewWithWriter(config, os.Stdout)
}

// NewWithWriter returns a Reporter that writes to w, which makes the output
// testable.
func NewWithWriter(config Config, w io.Writer) *Reporter {
	return &Reporter{config: config, out: w}
}

func (r *Reporter) Report(results *detector.ScanResults) error {
	if r.config.JSONOutput {
		return r.reportJSON(results)
	}
	return r.reportText(results)
}

type jsonReport struct {
	SchemaVersion int        `json:"schema_version"`
	TotalFiles    int        `json:"total_files"`
	DetectedFiles int        `json:"detected_files"`
	AverageScore  float64    `json:"average_score"`
	MLEnabled     bool       `json:"ml_enabled"`
	Threshold     float64    `json:"threshold"`
	Files         []jsonFile `json:"files"`
}

type jsonFile struct {
	Path     string       `json:"path"`
	Score    float64      `json:"score"`
	Language string       `json:"language"`
	Detected bool         `json:"detected"`
	Signals  []jsonSignal `json:"signals,omitempty"`
	Error    string       `json:"error,omitempty"`
}

type jsonSignal struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Class       signal.Class      `json:"class"`
	Scope       signal.Scope      `json:"scope"`
	Advisory    bool              `json:"advisory"`
	Status      signal.Status     `json:"status"`
	SkipReason  string            `json:"skip_reason,omitempty"`
	Score       float64           `json:"score"`
	Evidence    []signal.Evidence `json:"evidence,omitempty"`
}

func (r *Reporter) reportJSON(results *detector.ScanResults) error {
	output := jsonReport{
		SchemaVersion: jsonSchemaVersion,
		TotalFiles:    results.TotalFiles,
		DetectedFiles: results.DetectedFiles,
		AverageScore:  results.AverageScore,
		MLEnabled:     results.UsedML,
		Threshold:     r.config.Threshold,
		Files:         r.formatFilesForJSON(results.Files),
	}

	encoder := json.NewEncoder(r.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (r *Reporter) formatFilesForJSON(files []detector.FileResult) []jsonFile {
	output := make([]jsonFile, 0, len(files))

	for _, file := range files {
		fileData := jsonFile{
			Path:     file.Path,
			Score:    file.Score,
			Language: file.Language,
			Detected: file.Score >= r.config.Threshold,
		}

		if r.config.Verbose {
			// the skipped signals are included on purpose: "not evaluated" is
			// information a consumer needs, and it is not the same as clean.
			fileData.Signals = make([]jsonSignal, 0, len(file.Signals))
			for _, sig := range file.Signals {
				fileData.Signals = append(fileData.Signals, jsonSignal{
					ID:          sig.Meta.ID,
					Name:        sig.Meta.Name,
					Description: sig.Meta.Description,
					Class:       sig.Meta.Class,
					Scope:       sig.Meta.Scope,
					Advisory:    sig.Meta.Advisory,
					Status:      sig.Status,
					SkipReason:  sig.SkipReason,
					Score:       sig.Score,
					Evidence:    sig.Evidence,
				})
			}
		}

		if file.Error != nil {
			fileData.Error = file.Error.Error()
		}

		output = append(output, fileData)
	}

	return output
}

func (r *Reporter) printf(format string, a ...any) { fmt.Fprintf(r.out, format, a...) }
func (r *Reporter) println(a ...any)               { fmt.Fprintln(r.out, a...) }

func (r *Reporter) reportText(results *detector.ScanResults) error {
	r.println()
	r.println(heavyRule)
	r.println()
	r.printBanner()
	r.println()
	r.println(heavyRule)
	r.println()

	mode := "Heuristics Only"
	if results.UsedML {
		mode = "Machine Learning + Heuristics"
	}

	r.printf("Detection Mode:       %s\n", mode)
	r.printf("Total Files Scanned:  %d\n", results.TotalFiles)
	r.printf("Files Detected:       %d\n", results.DetectedFiles)
	r.printf("Average Score:        %.2f\n", results.AverageScore)
	r.printf("Detection Threshold:  %.2f\n", r.config.Threshold)
	r.println()

	// Highest score first; ties broken by path so output is deterministic.
	sortedFiles := make([]detector.FileResult, len(results.Files))
	copy(sortedFiles, results.Files)
	sort.SliceStable(sortedFiles, func(i, j int) bool {
		if sortedFiles[i].Score != sortedFiles[j].Score {
			return sortedFiles[i].Score > sortedFiles[j].Score
		}
		return sortedFiles[i].Path < sortedFiles[j].Path
	})

	var flagged, review []detector.FileResult
	reviewFloor := r.config.Threshold * reviewFraction
	for _, file := range sortedFiles {
		switch {
		case file.Score >= r.config.Threshold:
			flagged = append(flagged, file)
		case file.Score >= reviewFloor:
			review = append(review, file)
		}
	}

	if results.DetectedFiles > 0 {
		r.printSection("[!] HIGH-RISK FILES (above threshold):", flagged)
	}
	if r.config.Verbose {
		r.printSection("[?] REVIEW RECOMMENDED (below threshold but noteworthy):", review)
	}

	r.println(heavyRule)
	if results.DetectedFiles > 0 {
		r.printf("FAILED: %d file(s) require attention (risk threshold exceeded)\n", results.DetectedFiles)
	} else {
		r.println("PASSED: All files are within acceptable risk thresholds")
	}
	r.println("Note: Scores represent heuristic and/or ML-based risk estimates.")
	r.println("Review is recommended for flagged files.")
	r.println(heavyRule)

	return nil
}

func (r *Reporter) printBanner() {
	maxLen := 0
	for _, line := range logoLines {
		if l := utf8.RuneCountInString(line); l > maxLen {
			maxLen = l
		}
	}

	padding := 0
	if maxLen < lineWidth {
		padding = (lineWidth - maxLen) / 2
	}
	indent := strings.Repeat(" ", padding)

	for _, line := range logoLines {
		if line == "" {
			r.println()
		} else {
			r.println(indent + line)
		}
	}
}

func (r *Reporter) printSection(title string, files []detector.FileResult) {
	if len(files) == 0 {
		return
	}
	r.println(title)
	r.println(lightRule)
	for _, file := range files {
		r.printFileResult(file)
	}
	r.println()
}

func (r *Reporter) printFileResult(file detector.FileResult) {
	mlBadge := ""
	if file.UsedML {
		mlBadge = " [ML]"
	}

	r.printf("%s\n", file.Path)
	r.printf("   Score: %.2f%s | Language: %s\n", file.Score, mlBadge, file.Language)

	if file.Error != nil {
		r.printf("   Error: %s\n", file.Error.Error())
		return
	}

	if !r.config.Verbose {
		return
	}

	if file.UsedML {
		r.printf("   Confidence: %.2f | ML Score: %.2f | Heuristic Score: %.2f\n",
			file.Confidence, file.MLScore, file.HeuristicScore)
	}

	r.printSignals(file.Signals)
}

func (r *Reporter) printSignals(signals []signal.Signal) {
	var fired, skipped []signal.Signal
	for _, sig := range signals {
		switch {
		case sig.Status == signal.StatusSkipped:
			skipped = append(skipped, sig)
		case sig.Score > 0:
			fired = append(fired, sig)
		}
	}

	if len(fired) > 0 {
		sort.SliceStable(fired, func(i, j int) bool {
			return fired[i].Score > fired[j].Score
		})

		r.println("   Signals:")
		for _, sig := range fired {
			badge := ""
			if sig.Meta.Advisory {
				badge = " [advisory]"
			}
			r.printf("     • %s (%.2f)%s\n", sig.Meta.Name, sig.Score, badge)

			if len(sig.Evidence) == 0 {
				r.printf("       %s\n", sig.Meta.Description)
			} else {
				r.printEvidence(sig.Evidence)
			}
			r.println()
		}
	}

	if len(skipped) > 0 {
		parts := make([]string, 0, len(skipped))
		for _, sig := range skipped {
			if sig.SkipReason != "" {
				parts = append(parts, fmt.Sprintf("%s (%s)", sig.Meta.Name, sig.SkipReason))
			} else {
				parts = append(parts, sig.Meta.Name)
			}
		}
		r.printf("   Not evaluated: %s\n", strings.Join(parts, ", "))
	}
}

func (r *Reporter) printEvidence(evidence []signal.Evidence) {
	shown := evidence
	if len(shown) > maxEvidencePerSignal {
		shown = shown[:maxEvidencePerSignal]
	}

	for _, ev := range shown {
		loc := ""
		switch {
		case ev.Path != "" && ev.Line > 0:
			loc = fmt.Sprintf("%s:%d: ", ev.Path, ev.Line)
		case ev.Path != "":
			loc = ev.Path + ": "
		case ev.Line > 0:
			loc = fmt.Sprintf("line %d: ", ev.Line)
		}

		if text := strings.TrimSpace(loc + ev.Detail); text != "" {
			r.printf("       %s\n", text)
		}
		if len(ev.Metrics) > 0 {
			r.printf("       %s\n", formatMetrics(ev.Metrics))
		}
	}

	if extra := len(evidence) - len(shown); extra > 0 {
		r.printf("       ... and %d more\n", extra)
	}
}

// formatMetrics renders metrics in a stable (sorted) order.
func formatMetrics(m map[string]float64) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+": "+formatMetric(m[k]))
	}
	return strings.Join(parts, ", ")
}

// formatMetric prints whole numbers without decimals and everything else
// with two.
func formatMetric(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e9 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}
