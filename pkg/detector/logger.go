package detector

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"sentinel/pkg/ml"
	"sentinel/pkg/models"
)

// LogTrainingData saves scan results to a CSV file for Machine Learning training purposes.
func (d *Detector) LogTrainingData(results *ScanResults, isAI bool) error {
	trainingDataFile, err := os.OpenFile("sentinel_training.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer func() {
		if err := trainingDataFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close training data file: %v\n", err)
		}
	}()

	writer := csv.NewWriter(trainingDataFile)
	defer writer.Flush()

	fe := ml.NewFeatureExtractor()
	info, err := trainingDataFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat training file: %w", err)
	}

	if info.Size() == 0 {
		dummyVector := fe.Extract([]models.Signal{}, "unknown", ml.CodeMetrics{})
		header := append(dummyVector.Names, "label")
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	labelStr := "0"
	if isAI {
		labelStr = "1"
	}

	for _, f := range results.Files {
		if f.Error != nil {
			continue
		}
		content, err := os.ReadFile(f.Path)
		if err != nil {
			continue
		}
		metrics := ml.ExtractCodeMetrics(string(content))
		vector := fe.Extract(f.Signals, f.Language, metrics)
		row := make([]string, 0, len(vector.Features)+1)
		for _, val := range vector.Features {
			row = append(row, strconv.FormatFloat(float64(val), 'f', 4, 32))
		}
		row = append(row, labelStr)

		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}
