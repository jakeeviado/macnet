// Package analyzer runs the heuristic checks and returns their signals.
//
// # ADDING A SIGNAL
//
//  1. Write the check under analyzer/signals with its own SignalMeta. It
//     returns signal.NewSignal(meta, score, evidence...) when it ran, or
//     signal.Skipped(meta, reason) when it could not evaluate the input.
//
//  2. Add it to signals.FileChecks (file scope) or signals.DiffChecks
//     (diff scope) in analyzer/signals/registry.go.
//
//  3. File scope only: append its name to signalRegistry in
//     pkg/ml/features.go. NumFeatures follows automatically, but the vector
//     length changes, so the ONNX model must be retrained. Diff signals are
//     not part of the ML feature vector.
//
// The tests in this package fail if a file check is missing from the ML
// registry or if a check reports the wrong metadata.
//
// # PROFILES
//
// A check runs only if its Meta.MinProfile is at or below the analyzer's
// profile. Checks the profile disables are not run at all, so they are absent
// from the result (the ML vector sees them as Missing) and do not appear
// under "Not evaluated" in reports. Skipped is reserved for checks that were
// eligible but could not evaluate their input.
package analyzer

import (
	"macnet/pkg/analyzer/signals"
	"macnet/pkg/signal"
)

type Analyzer struct {
	profile signal.Profile
}

// New returns an Analyzer that runs the checks active in the given profile.
func New(profile signal.Profile) *Analyzer {
	return &Analyzer{profile: profile}
}

// AnalyzeFile runs every file-scope check active in the analyzer's profile
// against the provided code and returns one signal per check that ran.
func (a *Analyzer) AnalyzeFile(code string, language string) []signal.Signal {
	sigList := make([]signal.Signal, 0, len(signals.FileChecks))
	for _, c := range signals.FileChecks {
		if !c.Meta.ActiveIn(a.profile) {
			continue
		}
		sigList = append(sigList, c.Run(code))
	}
	return sigList
}
