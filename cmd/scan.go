package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"macnet/pkg/analyzer/signals"
	"macnet/pkg/detector"
	"macnet/pkg/reporter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	scanPath     string
	languages    []string
	threshold    float64
	failOnDetect bool
	excludePaths []string
	gitDiff      string
	trainingData bool
	label        string
	modelPath    string
	noML         bool
	mlOnly       bool
	mlWeight     float64
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan code for elevated-risk patterns",
	Long: `Scan source code files for patterns that may indicate elevated risk in AI-assisted workflows.

Examples:
  # Scan current directory
  macnet scan --path .

  # Scan with specific languages
  macnet scan --path ./src --languages python,java,javascript

  # Scan git diff against main/master branch
  macnet scan --git-diff origin/main --threshold 0.75

  # Fail CI build if high-risk patterns detected
  macnet scan --path . --fail-on-detection --threshold 0.8`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringVarP(&scanPath, "path", "p", ".", "path to scan")
	scanCmd.Flags().StringSliceVarP(&languages, "languages", "l", []string{}, "languages to scan (e.g., python,java,go)")
	scanCmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.7, "detection threshold (0.0-1.0)")
	scanCmd.Flags().BoolVar(&failOnDetect, "fail-on-detection", false, "exit with error code if high-risk patterns are detected")
	scanCmd.Flags().StringSliceVar(&excludePaths, "exclude", []string{}, "paths to exclude from scanning")
	scanCmd.Flags().StringVar(&gitDiff, "git-diff", "", "scan only files changed in git diff against specified branch")
	scanCmd.Flags().BoolVar(&trainingData, "collect", false, "save scan results to local training CSV")
	scanCmd.Flags().StringVar(&label, "label", "", "label for training data: 'flagged' or 'human'")
	scanCmd.Flags().StringVar(&modelPath, "model", "", "path to ONNX model file")
	scanCmd.Flags().BoolVar(&noML, "no-ml", false, "disable ML inference (heuristics only)")
	scanCmd.Flags().BoolVar(&mlOnly, "ml-only", false, "use ML only (fail if model not available)")
	scanCmd.Flags().Float64Var(&mlWeight, "ml-weight", 0.7, "weight given to ML score (0.0-1.0)")
}

func runScan(cmd *cobra.Command, args []string) error {
	verbose := viper.GetBool("verbose")
	jsonOutput := viper.GetBool("json")

	if verbose {
		fmt.Fprintf(os.Stderr, "Scanning path: %s\n", scanPath)
		fmt.Fprintf(os.Stderr, "Threshold: %.2f\n", threshold)
		if len(languages) > 0 {
			fmt.Fprintf(os.Stderr, "Languages: %v\n", languages)
		}
	}

	det := detector.New(detector.DetectorConfiguration{
		Threshold:    threshold,
		Languages:    languages,
		ExcludePaths: excludePaths,
		IsVerbose:    verbose,
		ModelPath:    modelPath,
		UseML:        !noML && modelPath != "",
		MLWeight:     mlWeight,
		IsMLOnly:     mlOnly,
	})

	var results *detector.ScanResults
	var hasDetections bool

	if gitDiff != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "Analyzing git diff against: %s\n", gitDiff)
		}

		fileDiffs, diffStats, err := getGitDiffContext(gitDiff)
		if err != nil {
			return fmt.Errorf("failed to get git diff context: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Diff stats: +%d, -%d across %d files\n",
				diffStats.Additions, diffStats.Deletions, diffStats.FilesChanged)
		}

		diffResults, err := det.ScanDiff(fileDiffs, diffStats)
		if err != nil {
			return fmt.Errorf("git diff scan failed: %w", err)
		}

		hasDetections = diffResults.IsFlagged

		var fileResults []detector.FileResult
		for _, fd := range fileDiffs {
			fileResults = append(fileResults, detector.FileResult{
				Path:           fd.Path,
				Signals:        diffResults.Signals,
				HeuristicScore: diffResults.FinalScore,
				Score:          diffResults.FinalScore,
			})
		}

		detectedCount := 0
		if diffResults.IsFlagged {
			detectedCount = len(fileResults)
		}

		results = &detector.ScanResults{
			Files:         fileResults,
			TotalFiles:    len(fileResults),
			AverageScore:  diffResults.FinalScore,
			DetectedFiles: detectedCount,
			UsedML:        false,
		}

	} else {
		files, err := collectFiles(scanPath, excludePaths)
		if err != nil {
			return fmt.Errorf("failed to collect files: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Found %d files to scan\n", len(files))
		}

		results, err = det.ScanFiles(files)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		if trainingData {
			if label != "flagged" && label != "human" {
				return fmt.Errorf("please specify --label as 'flagged' or 'human' when using --collect")
			}
			isFlagged := (label == "flagged")
			if err := det.LogTrainingData(results, isFlagged); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to log training data: %v\n", err)
			} else {
				fmt.Println("Successfully logged results to macnet_training.csv")
			}
		}

		hasDetections = results.HasDetections(threshold)
	}

	// REPORT RESULTS
	rep := reporter.New(reporter.Config{
		JSONOutput: jsonOutput,
		Verbose:    verbose,
		Threshold:  threshold,
	})

	if err := rep.Report(results); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Fail the build if detections exceed the configured threshold.
	if failOnDetect && hasDetections {
		return fmt.Errorf("high-risk code detected above threshold %.2f", threshold)
	}

	return nil
}

func collectFiles(root string, exclude []string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// SKIP EXCLUDED PATHS
		for _, ex := range exclude {
			if matched, _ := filepath.Match(ex, path); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// SKIP COMMON DIRECTORIES
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == "target" || name == "build" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		// ADD SUPPORTED SOURCE SILES
		if isSupportedFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func isSupportedFile(path string) bool {
	ext := filepath.Ext(path)
	supportedExts := map[string]bool{
		".py":    true, // Python
		".java":  true, // Java
		".js":    true, // JavaScript
		".ts":    true, // TypeScript
		".go":    true, // Go
		".rs":    true, // Rust
		".cpp":   true, // C++
		".c":     true, // C
		".h":     true, // C/C++ headers
		".rb":    true, // Ruby
		".php":   true, // PHP
		".cs":    true, // C#
		".kt":    true, // Kotlin
		".swift": true, // Swift
	}
	return supportedExts[ext]
}

func getGitDiffContext(branch string) ([]signals.FileDiff, signals.DiffStats, error) {
	nameCmd := exec.Command("git", "diff", "--name-only", branch)
	nameOutput, err := nameCmd.Output()
	if err != nil {
		return nil, signals.DiffStats{}, fmt.Errorf("git diff name-only failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(nameOutput)), "\n")
	var fileDiffs []signals.FileDiff

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if _, err := os.Stat(line); err == nil && isSupportedFile(line) {
			contentBytes, err := os.ReadFile(line)
			if err != nil {
				continue
			}
			fileDiffs = append(fileDiffs, signals.FileDiff{
				Path:    line,
				Content: string(contentBytes),
			})
		}
	}

	statCmd := exec.Command("git", "diff", "--shortstat", branch)
	statOutput, err := statCmd.Output()
	stats := signals.DiffStats{FilesChanged: len(fileDiffs)}

	if err == nil {
		statStr := string(statOutput)
		_, _ = fmt.Sscanf(statStr, " %d files changed, %d insertions(+), %d deletions(-)",
			&stats.FilesChanged, &stats.Additions, &stats.Deletions)
		if stats.FilesChanged == 0 {
			_, _ = fmt.Sscanf(statStr, " %d file changed, %d insertions(+), %d deletions(-)",
				&stats.FilesChanged, &stats.Additions, &stats.Deletions)
		}
	}
	return fileDiffs, stats, nil
}
