package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"macnet/pkg/detector"
	"macnet/pkg/models"
)

type Config struct {
	JSONOutput bool
	Verbose    bool
	Threshold  float64
}

type Reporter struct {
	config Config
}

func New(config Config) *Reporter {
	return &Reporter{config: config}
}

func (r *Reporter) Report(results *detector.ScanResults) error {
	if r.config.JSONOutput {
		return r.reportJSON(results)
	}
	return r.reportText(results)
}

func (r *Reporter) reportJSON(results *detector.ScanResults) error {
	output := map[string]interface{}{
		"total_files":    results.TotalFiles,
		"detected_files": results.DetectedFiles,
		"average_score":  results.AverageScore,
		"ml_enabled":     results.UsedML,
		"threshold":      r.config.Threshold,
		"files":          r.formatFilesForJSON(results.Files),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (r *Reporter) formatFilesForJSON(files []detector.FileResult) []map[string]interface{} {
	output := make([]map[string]interface{}, 0, len(files))

	for _, file := range files {
		fileData := map[string]interface{}{
			"path":     file.Path,
			"score":    file.Score,
			"language": file.Language,
			"detected": file.Score >= r.config.Threshold,
		}

		if r.config.Verbose {
			signals := make([]map[string]interface{}, 0, len(file.Signals))
			for _, sig := range file.Signals {
				signals = append(signals, map[string]interface{}{
					"name":        sig.Name,
					"score":       sig.Score,
					"description": sig.Description,
					"evidence":    sig.Evidence,
				})
			}
			fileData["signals"] = signals
		}

		if file.Error != nil {
			fileData["error"] = file.Error.Error()
		}

		output = append(output, fileData)
	}

	return output
}

func (r *Reporter) reportText(results *detector.ScanResults) error {
	fmt.Println()
	fmt.Println("░▄█ ▀▀▄ ▀▀█  ▀▀▄ ▀▀█  ▀▀▀")
	fmt.Println("█ ░ ░ ▓ ░    █ ▓ ░     ░")
	fmt.Println("▓ ▒ ▒▀▒ ▒    ▓ ▒ ▒▀    ▒")
	fmt.Println("▒ ▓ ▓ ░ ▓ ▄  ▒ ░ ▓ ▄   ▓")
	fmt.Println("▀ ▀ ▀ ▀ ▀▀▀  ░ ▀ ▀▀█   ░")
	fmt.Println("        ▀▀▀   ▀▀▀ ▀▀▀")
	fmt.Println("Analysis Report")
	fmt.Println("================================================================================")
	fmt.Println()

	mode := "Heuristics Only"
	if results.UsedML {
		mode = "Machine Learning + Heuristics"
	}

	fmt.Printf("Detection Mode:       %s\n", mode)
	fmt.Printf("Total Files Scanned:  %d\n", results.TotalFiles)
	fmt.Printf("Files Detected:       %d\n", results.DetectedFiles)
	fmt.Printf("Average Score:        %.2f\n", results.AverageScore)
	fmt.Printf("Detection Threshold:  %.2f\n", r.config.Threshold)
	fmt.Println()

	sortedFiles := make([]detector.FileResult, len(results.Files))
	copy(sortedFiles, results.Files)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Score > sortedFiles[j].Score
	})

	if results.DetectedFiles > 0 {
		fmt.Println("[!] HIGH-RISK FILES (above threshold):")
		fmt.Println("--------------------------------------------------------------------------------")

		for _, file := range sortedFiles {
			if file.Score >= r.config.Threshold {
				r.printFileResult(file)
			}
		}
		fmt.Println()
	}

	suspiciousFiles := 0
	for _, file := range sortedFiles {
		if file.Score >= r.config.Threshold*0.6 && file.Score < r.config.Threshold {
			suspiciousFiles++
		}
	}

	if suspiciousFiles > 0 && r.config.Verbose {
		fmt.Println("[?] REVIEW RECOMMENDED (below threshold but noteworthy):")
		fmt.Println("--------------------------------------------------------------------------------")

		for _, file := range sortedFiles {
			if file.Score >= r.config.Threshold*0.6 && file.Score < r.config.Threshold {
				r.printFileResult(file)
			}
		}
		fmt.Println()
	}

	fmt.Println("================================================================================")
	if results.DetectedFiles > 0 {
		fmt.Printf("FAILED: %d file(s) require attention (risk threshold exceeded)\n", results.DetectedFiles)
	} else {
		fmt.Println("PASSED: All files are within acceptable risk thresholds")
	}
	fmt.Println("Note: Scores represent heuristic and/or ML-based risk estimates. Review is recommended for flagged files.")
	fmt.Println("================================================================================")

	return nil
}

func (r *Reporter) printFileResult(file detector.FileResult) {
	mlBadge := ""
	if file.UsedML {
		mlBadge = " [ML]"
	}

	fmt.Printf("%s\n", file.Path)
	fmt.Printf("   Score: %.2f%s | Language: %s\n", file.Score, mlBadge, file.Language)

	if file.Error != nil {
		fmt.Printf("   Error: %s\n", file.Error.Error())
		return
	}

	if r.config.Verbose {
		if file.UsedML {
			fmt.Printf("   Confidence: %.2f | ML Score: %.2f | Heuristic Score: %.2f\n",
				file.Confidence, file.MLScore, file.HeuristicScore)
		}

		if len(file.Signals) > 0 {
			fmt.Println("   Signals:")
			signals := make([]models.Signal, len(file.Signals))
			copy(signals, file.Signals)
			sort.Slice(signals, func(i, j int) bool {
				return signals[i].Score > signals[j].Score
			})

			for _, sig := range signals {
				if sig.Score > 0.0 {
					fmt.Printf("     • %s (%.2f): %s\n", sig.Name, sig.Score, sig.Description)
					if sig.Evidence != "" {
						fmt.Printf("       %s\n", sig.Evidence)
					}
				}
			}
		}
	}
}
