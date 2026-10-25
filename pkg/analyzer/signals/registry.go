package signals

import "macnet/pkg/signal"

// FileCheck pairs a file-scope check with its metadata, so the analyzer can
// decide whether it applies (profile, scope) before running it.
type FileCheck struct {
	Meta signal.SignalMeta
	Run  func(code string) signal.Signal
}

// DiffContext is the input to every diff-scope check. A new diff check reads
// what it needs from here, so adding one never changes a function signature.
type DiffContext struct {
	Files []FileDiff
	Stats DiffStats
}

// DiffCheck pairs a diff-scope check with its metadata.
type DiffCheck struct {
	Meta signal.SignalMeta
	Run  func(DiffContext) signal.Signal
}

// FileChecks lists every file-scope check. Register new file checks here,
// and also append their name to signalRegistry in pkg/ml/features.go.
var FileChecks = []FileCheck{
	{commentDensityMeta, CheckCommentDensity},
	{genericNamingMeta, CheckGenericNaming},
	{repetitivePatternsMeta, CheckRepetitivePatterns},
	{codeComplexityMeta, CheckCodeComplexity},
	{formattingConsistencyMeta, CheckFormattingConsistency},
	{commentRedundancyMeta, CheckCommentRedundancy},
	{emojiSentimentMeta, CheckEmojiSentiment},
	{identifierOrderMeta, CheckIdentifierOrder},
	{defensiveRatioMeta, CheckDefensiveRatio},
}

// DiffChecks lists every diff-scope check. Diff signals are not part of the
// ML feature vector.
var DiffChecks = []DiffCheck{
	{styleDriftMeta, func(c DiffContext) signal.Signal { return CheckStyleDrift(c.Files) }},
	{additionDeletionMeta, func(c DiffContext) signal.Signal { return CheckAdditionDeletionRatio(c.Stats) }},
}
