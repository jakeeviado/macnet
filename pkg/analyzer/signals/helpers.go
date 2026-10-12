package signals

import "fmt"

// FormatEvidence formats evidence strings for detection signals.
func FormatEvidence(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
