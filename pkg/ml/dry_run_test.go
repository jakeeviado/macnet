package ml

import (
	"fmt"
	"macnet/pkg/analyzer"
	"os"
	"path/filepath"
	"testing"
)

// Helper to visually inspect the Feature Vector.
// It helps to verify that the Registry order matches the Machine Learning Model's expectations.
func RunDryRun(code string, language string) {
	config := DetectionConfig{
		ModelPath:     "model.onnx",
		MLWeight:      0.7,
		FallbackMode:  true,
		MinConfidence: 0.5,
		NumFeatures:   21,
	}
	det, err := NewMLDetector(config)
	if err != nil {
		fmt.Printf("Error initializing detector: %v\n", err)
		return
	}

	defer func() {
		if err := det.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close detector: %v\n", err)
		}
	}()

	az := analyzer.New()
	signals := az.AnalyzeFile(code, language)

	result, err := det.Detect(signals, language, code)
	if err != nil {
		fmt.Printf("Detection error: %v\n", err)
		return
	}

	metrics := ExtractCodeMetrics(code)
	vector := det.featureExtractor.Extract(signals, language, metrics)

	fmt.Println("\n================================================================================")
	fmt.Printf("                ⌀ MACnet - FULL ML PIPELINE DRY RUN\n")
	fmt.Println("================================================================================")
	fmt.Printf("  LANGUAGE: %-10s | LINES: %-5d | CHARS: %-5d\n",
		language, metrics.LineCount, metrics.CharCount)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("  %-7s | %-28s | %-12s\n", "INDEX", "FEATURE NAME", "VALUE")
	fmt.Println("  --------|------------------------------|--------------")

	for i := 0; i < len(vector.Features); i++ {
		status := ""
		if vector.Features[i] > 0.0 && i <= 10 {
			status = " [SIGNAL]"
		}
		if vector.Features[i] > 0.5 {
			status = " <!!> TARGET"
		}
		fmt.Printf("  [%02d]    | %-28s | %-12.4f%s\n",
			i, vector.Names[i], vector.Features[i], status)
	}

	fmt.Println("--------------------------------------------------------------------------------")

	fmt.Printf("  HEURISTIC RAW   : %.4f\n", result.HeuristicScore)
	fmt.Printf("  ML MODEL PROB   : %.4f (Used ML: %v)\n", result.MLScore, result.UsedML)
	fmt.Printf("  FINAL AGGREGATE : %.4f\n", result.FinalScore)
	fmt.Printf("  CONFIDENCE      : %02.1f%%\n", result.Confidence*100)

	verdict := "LOW RISK"
	if result.IsHighRisk {
		verdict = "FLAGGED FOR REVIEW"
	}
	fmt.Printf("  VERDICT         : %s\n", verdict)
	fmt.Println("================================================================================")
}

// TestFeatureExtractionPipeline runs the dry run on samples provided in directory.
// Run this with: go test -v ./pkg/ml -run TestFeatureExtractionPipeline
func TestFeatureExtractionPipeline(t *testing.T) {
	examplesDir := "../../examples"
	err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name()[0] == '.' {
			return nil
		}
		t.Run(info.Name(), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", path, err)
			}

			ext := filepath.Ext(path)
			lang := "unknown"

			switch ext {
			case ".py":
				lang = "python"
			case ".go":
				lang = "go"
			case ".java":
				lang = "java"
			case ".js", ".ts":
				lang = "javascript"
			case ".rs":
				lang = "rust"
			case ".cpp", ".c", ".h", ".cc":
				lang = "cpp"
			case ".rb":
				lang = "ruby"
			case ".php":
				lang = "php"
			case ".cs":
				lang = "csharp"
			case ".kt":
				lang = "kotlin"
			case ".swift":
				lang = "swift"
			}

			fmt.Printf("\n>>> FILE: %s <<<\n", path)
			RunDryRun(string(content), lang)
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Directory walk failed: %v", err)
	}
}
