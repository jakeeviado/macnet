package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"macnet/pkg/analyzer/signals"
	"macnet/pkg/detector"
	"macnet/pkg/reporter"
	"macnet/pkg/signal"

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
	profileName  string
)

// defaultThreshold is used when --threshold is not given explicitly.
// Both profiles start at the previous global default. Set strict from a
// measured false-positive rate on known-human code, not from a guess.
var defaultThreshold = map[signal.Profile]float64{
	signal.ProfileLoose:  0.70,
	signal.ProfileStrict: 0.70,
}

var supportedExts = map[string]bool{
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

var skippedDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"target": true, "build": true, "dist": true,
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan code for elevated-risk patterns",
	Long: `Scan source code files for patterns that may indicate elevated risk in AI-assisted workflows.

Profiles:
  loose   file-level signals plus cheap diff signals (default)
  strict  adds repo-baseline, history and author-baseline signals

Examples:
  # Scan current directory
  macnet scan --path .

  # Scan with specific languages
  macnet scan --path ./src --languages python,java,javascript

  # Scan the changes on this branch against main
  macnet scan --git-diff origin/main --threshold 0.75

  # Strict profile for assessments (needs full git history in CI)
  macnet scan --git-diff origin/main --profile strict

  # Fail CI build if high-risk patterns detected
  macnet scan --path . --fail-on-detection --threshold 0.8`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringVarP(&scanPath, "path", "p", ".", "path to scan")
	scanCmd.Flags().StringSliceVarP(&languages, "languages", "l", []string{}, "languages to scan (e.g., python,java,go)")
	scanCmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.7, "detection threshold (0.0-1.0); defaults per profile")
	scanCmd.Flags().BoolVar(&failOnDetect, "fail-on-detection", false, "exit with error code if high-risk patterns are detected")
	scanCmd.Flags().StringSliceVar(&excludePaths, "exclude", []string{}, "paths or patterns to exclude from scanning (e.g., vendor, *.pb.go, src/gen)")
	scanCmd.Flags().StringVar(&gitDiff, "git-diff", "", "scan only files changed since the merge-base with the given branch")
	scanCmd.Flags().BoolVar(&trainingData, "collect", false, "save scan results to local training CSV")
	scanCmd.Flags().StringVar(&label, "label", "", "label for training data: 'flagged' or 'human'")
	scanCmd.Flags().StringVar(&modelPath, "model", "", "path to ONNX model file")
	scanCmd.Flags().BoolVar(&noML, "no-ml", false, "disable ML inference (heuristics only)")
	scanCmd.Flags().BoolVar(&mlOnly, "ml-only", false, "use ML only (fail if model not available)")
	scanCmd.Flags().Float64Var(&mlWeight, "ml-weight", 0.7, "weight given to ML score (0.0-1.0)")
	scanCmd.Flags().StringVar(&profileName, "profile", "loose", "detection profile: 'loose' or 'strict'")

	scanCmd.MarkFlagsMutuallyExclusive("no-ml", "ml-only")

	// Lets `profile: strict` in .macnet.yaml work; the flag still wins.
	_ = viper.BindPFlag("profile", scanCmd.Flags().Lookup("profile"))
}

// parseProfile converts the flag value into a signal.Profile.
func parseProfile(name string) (signal.Profile, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "loose":
		return signal.ProfileLoose, nil
	case "strict":
		return signal.ProfileStrict, nil
	default:
		return 0, fmt.Errorf("invalid --profile %q: must be 'loose' or 'strict'", name)
	}
}

// validateFlags fails fast, before any scanning work happens.
func validateFlags() (signal.Profile, error) {
	profile, err := parseProfile(viper.GetString("profile"))
	if err != nil {
		return 0, err
	}
	if threshold < 0 || threshold > 1 {
		return 0, fmt.Errorf("--threshold must be between 0.0 and 1.0, got %.2f", threshold)
	}
	if mlWeight < 0 || mlWeight > 1 {
		return 0, fmt.Errorf("--ml-weight must be between 0.0 and 1.0, got %.2f", mlWeight)
	}
	if mlOnly && modelPath == "" {
		return 0, fmt.Errorf("--ml-only requires --model")
	}
	if trainingData {
		if label != "flagged" && label != "human" {
			return 0, fmt.Errorf("please specify --label as 'flagged' or 'human' when using --collect")
		}
		if gitDiff != "" {
			return 0, fmt.Errorf("--collect is not supported together with --git-diff (training data is file-level)")
		}
	}
	return profile, nil
}

func runScan(cmd *cobra.Command, args []string) error {
	verbose := viper.GetBool("verbose")
	jsonOutput := viper.GetBool("json")

	profile, err := validateFlags()
	if err != nil {
		return err
	}
	// Flags are valid from here on, so runtime failures (including
	// --fail-on-detection) should not dump the usage text.
	cmd.SilenceUsage = true

	if !cmd.Flags().Changed("threshold") {
		threshold = defaultThreshold[profile]
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Scanning path: %s\n", scanPath)
		fmt.Fprintf(os.Stderr, "Profile: %s\n", strings.ToLower(profileName))
		fmt.Fprintf(os.Stderr, "Threshold: %.2f\n", threshold)
		if len(languages) > 0 {
			fmt.Fprintf(os.Stderr, "Languages: %v\n", languages)
		}
	}

	det, err := detector.New(detector.DetectorConfiguration{
		Threshold:    threshold,
		Languages:    languages,
		ExcludePaths: excludePaths,
		IsVerbose:    verbose,
		ModelPath:    modelPath,
		UseML:        !noML && modelPath != "",
		MLWeight:     mlWeight,
		IsMLOnly:     mlOnly,
		Profile:      profile,
	})
	if err != nil {
		return err
	}

	var results *detector.ScanResults

	if gitDiff != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "Analyzing git diff against: %s\n", gitDiff)
		}
		if profile == signal.ProfileStrict && isShallowRepo() {
			fmt.Fprintln(os.Stderr, "Warning: shallow clone detected; history-based signals will be skipped (use full history, e.g. fetch-depth: 0)")
		}

		fileDiffs, diffStats, err := getGitDiffContext(gitDiff, excludePaths)
		if err != nil {
			return fmt.Errorf("failed to get git diff context: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Diff stats: +%d, -%d across %d files\n",
				diffStats.Additions, diffStats.Deletions, diffStats.FilesChanged)
		}

		if len(fileDiffs) == 0 {
			fmt.Fprintln(os.Stderr, "No supported files changed; nothing to scan")
			results = &detector.ScanResults{}
		} else {
			paths := make([]string, 0, len(fileDiffs))
			for _, fd := range fileDiffs {
				paths = append(paths, fd.Path)
			}

			// Per-file results come from the normal file-level scan, so each
			// file keeps its own score instead of sharing one diff-wide score.
			results, err = det.ScanFiles(paths)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Diff-level signals are reported separately and are advisory.
			diffResult, err := det.ScanDiff(fileDiffs, diffStats)
			if err != nil {
				return fmt.Errorf("git diff scan failed: %w", err)
			}
			results.DiffSignals = diffResult.Signals
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
			isFlagged := label == "flagged"
			if err := det.LogTrainingData(results, isFlagged); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to log training data: %v\n", err)
			} else {
				// stderr, so --json output on stdout stays valid
				fmt.Fprintln(os.Stderr, "Successfully logged results to macnet_training.csv")
			}
		}
	}

	// Only file-level results gate the build. Diff-level signals are
	// advisory until their false-positive rates have been measured.
	hasDetections := results.HasDetections(threshold)

	rep := reporter.New(reporter.Config{
		JSONOutput: jsonOutput,
		Verbose:    verbose,
		Threshold:  threshold,
	})

	if err := rep.Report(results); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	if failOnDetect && hasDetections {
		return fmt.Errorf("high-risk code detected above threshold %.2f", threshold)
	}

	return nil
}

// ---------------------------------------------------------------------------
// File collection
// ---------------------------------------------------------------------------

func collectFiles(root string, exclude []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root {
				return err // unreadable or missing root is fatal
			}
			fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", p, err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if p != root && isExcluded(p, exclude) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if p != root && skippedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if isSupportedFile(p) {
			files = append(files, p)
		}
		return nil
	})

	return files, err
}

// isExcluded reports whether name matches any --exclude entry. An entry
// matches if it is the path itself, a parent directory of it, a glob that
// matches the whole path, or (when it has no slash) a glob that matches any
// single path segment, so `--exclude node_modules` works at any depth.
func isExcluded(name string, patterns []string) bool {
	p := filepath.ToSlash(filepath.Clean(name))
	segments := strings.Split(p, "/")

	for _, pat := range patterns {
		pat = strings.TrimSpace(pat)
		if pat == "" {
			continue
		}
		pat = filepath.ToSlash(filepath.Clean(pat))
		if pat == "." {
			continue
		}

		if p == pat || strings.HasPrefix(p, pat+"/") {
			return true
		}
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
		if !strings.Contains(pat, "/") {
			for _, seg := range segments {
				if ok, _ := path.Match(pat, seg); ok {
					return true
				}
			}
		}
	}
	return false
}

func isSupportedFile(p string) bool {
	return supportedExts[strings.ToLower(filepath.Ext(p))]
}

// ---------------------------------------------------------------------------
// Git
// ---------------------------------------------------------------------------

// git runs a git command and returns trimmed stdout. On failure the error
// includes git's own stderr message.
func git(args ...string) (string, error) {
	c := exec.Command("git", args...)
	var stderr bytes.Buffer
	c.Stderr = &stderr

	out, err := c.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func isShallowRepo() bool {
	out, err := git("rev-parse", "--is-shallow-repository")
	return err == nil && out == "true"
}

// getGitDiffContext diffs HEAD against the merge-base with base (the
// three-dot form), so changes that landed on base after this branch forked
// do not show up as if this branch had reverted them.
//
// Each FileDiff holds only the lines this change ADDED, not the whole file,
// so diff signals judge the change and not the code that was already there.
// Paths are relative to the current directory, and only files under it are
// included. Uncommitted working-tree changes are not part of the diff.
func getGitDiffContext(base string, exclude []string) ([]signals.FileDiff, signals.DiffStats, error) {
	out, err := git(
		"-c", "core.quotepath=off",
		"diff", "-U0", "--no-color", "--no-ext-diff",
		"--diff-filter=ACMR", "--relative",
		base+"...HEAD",
	)
	if err != nil {
		return nil, signals.DiffStats{}, err
	}
	return parseUnifiedDiff(out, exclude)
}

// parseUnifiedDiff extracts per-file added lines and overall add/delete
// counts from `git diff -U0` output, keeping only supported, non-excluded
// source files. Statistics cover those files only, so lockfiles, vendored
// code, and docs do not skew the addition/deletion ratio.
func parseUnifiedDiff(diff string, exclude []string) ([]signals.FileDiff, signals.DiffStats, error) {
	var (
		files []signals.FileDiff
		stats signals.DiffStats

		current  string
		added    []string
		keep     bool
		inHunk   bool
		fileAdds int
		fileDels int
	)

	flush := func() {
		if keep && current != "" {
			files = append(files, signals.FileDiff{
				Path:    current,
				Content: strings.Join(added, "\n"),
			})
			stats.Additions += fileAdds
			stats.Deletions += fileDels
			stats.FilesChanged++
		}
		current, added, keep, inHunk, fileAdds, fileDels = "", nil, false, false, 0, 0
	}

	sc := bufio.NewScanner(strings.NewReader(diff))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()

		// File header. Only valid before the first hunk, because inside a
		// hunk a line starting with "+++" is just an added line.
		case !inHunk && strings.HasPrefix(line, "+++ "):
			name := strings.TrimSpace(line[4:])
			if name != "/dev/null" {
				current = strings.TrimPrefix(name, "b/")
				keep = isSupportedFile(current) && !isExcluded(current, exclude)
			}

		case strings.HasPrefix(line, "@@"):
			inHunk = true

		case inHunk && strings.HasPrefix(line, "+"):
			added = append(added, line[1:])
			fileAdds++

		case inHunk && strings.HasPrefix(line, "-"):
			fileDels++
		}
	}
	flush()

	return files, stats, sc.Err()
}
