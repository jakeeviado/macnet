package detector

import (
	"fmt"
	"os"
	"sync"

	"macnet/pkg/models"
)

// Concurrently scans a slice of target file paths using a bounded worker pool
// and returns aggregated scan results and statistics.
func (d *Detector) ScanFiles(files []string) (*ScanResults, error) {
	report := &ScanResults{
		Files:      make([]FileResult, 0, len(files)),
		TotalFiles: len(files),
		UsedML:     d.mlDetector != nil,
	}

	resultsChannel := make(chan FileResult, len(files))
	var waitGroup sync.WaitGroup
	semaphore := make(chan struct{}, 10)

	for _, file := range files {
		waitGroup.Add(1)
		go func(path string) {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			result := d.scanFile(path)
			resultsChannel <- result
		}(file)
	}

	go func() {
		waitGroup.Wait()
		close(resultsChannel)
	}()

	totalScore := 0.0
	for result := range resultsChannel {
		report.Files = append(report.Files, result)
		totalScore += result.Score

		if result.Score >= d.detectorConfig.Threshold {
			report.DetectedFiles++
		}
	}

	if len(report.Files) > 0 {
		report.AverageScore = totalScore / float64(len(report.Files))
	}

	return report, nil
}

// Processes a single target file, running heuristic analysis and evaluating
// machine learning predictions if an ML model is active.
func (d *Detector) scanFile(path string) FileResult {
	result := FileResult{
		Path:    path,
		Signals: make([]models.Signal, 0),
	}

	result.Language = detectLanguage(path)

	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 || info.IsDir() {
		result.Error = fmt.Errorf("invalid file: %w", err)
		return result
	}

	fileContent, err := os.ReadFile(path)
	if err != nil {
		result.Error = err
		return result
	}

	codeString := string(fileContent)

	heuristicSignals := d.analyzer.Analyze(codeString, result.Language)
	result.Signals = heuristicSignals
	hScore := calculateHeuristicMaxScore(heuristicSignals)
	result.HeuristicScore = hScore

	if d.mlDetector != nil {
		mlResult, err := d.mlDetector.Detect(heuristicSignals, result.Language, codeString)
		if err != nil {
			if d.detectorConfig.IsMLOnly {
				result.Error = fmt.Errorf("ML inference failed: %w", err)
				return result
			}
			result.Score = hScore
			result.UsedML = false
		} else {
			result.Score = mlResult.FinalScore
			result.MLScore = mlResult.MLScore
			result.UsedML = mlResult.UsedML
			result.Confidence = mlResult.Confidence
		}
	} else {
		result.Score = hScore
		result.UsedML = false
	}

	return result
}

// Evaluates a list of heuristic signals and returns the highest individual risk score.
func calculateHeuristicMaxScore(signals []models.Signal) float64 {
	if len(signals) == 0 {
		return 0.0
	}

	maxScore := 0.0
	for _, signal := range signals {
		if signal.Score > maxScore {
			maxScore = signal.Score
		}
	}
	return maxScore
}
