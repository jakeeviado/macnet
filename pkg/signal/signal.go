package signal

// Scope is what a signal analyzes.
type Scope string

// Signal analyzes this two types
const (
	ScopeFile Scope = "file"
	ScopeDiff Scope = "diff"
)

// Class is what a signal measures.
type Class string

const (
	ClassLexical    Class = "lexical"
	ClassStylistic  Class = "stylistic"
	ClassStructural Class = "structural"
	ClassProvenance Class = "provenance"
	ClassDependency Class = "dependency"
	ClassProcess    Class = "process"
)

// Tier is the data a signal needs. Higher tiers need more.
type Tier int

const (
	TierFile    Tier = iota // T0: one file's content
	TierDiff                // T1: the diff only
	TierRepo                // T2: repo baseline
	TierHistory             // T3: git history
	TierAuthor              // T4: author baseline
)

// Profile is a detection strictness level. Ordered: strict includes loose.
type Profile int

const (
	ProfileLoose Profile = iota
	ProfileStrict
)

// Status says whether a signal actually ran.
type Status string

const (
	StatusApplied Status = "applied"
	StatusSkipped Status = "skipped"
)

// SignalMeta is the static definition of a signal, declared once.
type SignalMeta struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Scope       Scope   `json:"scope"`
	Class       Class   `json:"class"`
	Tier        Tier    `json:"tier"`
	MinProfile  Profile `json:"min_profile"` // lowest profile that runs it
	Advisory    bool    `json:"advisory"`    // warn-only, never gates a build
	Weight      float64 `json:"weight"`      // contribution to the blended score
}

// ActiveIn reports whether the signal runs under profile p.
func (m SignalMeta) ActiveIn(p Profile) bool { return p >= m.MinProfile }

// Evidence is one concrete finding behind a signal.
type Evidence struct {
	Path    string             `json:"path,omitempty"`
	Line    int                `json:"line,omitempty"`
	Detail  string             `json:"detail"`
	Metrics map[string]float64 `json:"metrics,omitempty"` // e.g. generic_name_count: 47
}

// Signal is the result of running one SignalMeta on one target.
type Signal struct {
	Meta       SignalMeta `json:"meta"`
	Status     Status     `json:"status"`
	SkipReason string     `json:"skip_reason,omitempty"`
	Score      float64    `json:"score"` // normalized 0.0-1.0; meaningless when skipped
	Evidence   []Evidence `json:"evidence,omitempty"`
}

// Skipped builds a result for a signal that couldn't run.
func Skipped(m SignalMeta, reason string) Signal {
	return Signal{Meta: m, Status: StatusSkipped, SkipReason: reason}
}

// NewSignal builds an applied result and clamps the score to [0, 1].
func NewSignal(m SignalMeta, score float64, ev ...Evidence) Signal {
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return Signal{Meta: m, Status: StatusApplied, Score: score, Evidence: ev}
}
