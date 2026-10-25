package signals

import (
	"fmt"
	"macnet/pkg/signal"
	"strings"
)

var emojiSentimentMeta = signal.SignalMeta{
	ID:          "F7",
	Name:        "emoji_sentiment",
	Description: "Presence of non-standard emojis",
	Scope:       signal.ScopeFile,
	Class:       signal.ClassStylistic,
	Tier:        signal.TierFile,
	MinProfile:  signal.ProfileLoose,
	Weight:      1.0,
}

var aiHypeEmojis = []string{"🚀", "✨", "✅", "💡", "🛠️", "🤖"}

// CheckEmojiSentiment scans for informal emojis, which are an uncommon
// pattern in production codebases.
func CheckEmojiSentiment(code string) signal.Signal {
	if strings.TrimSpace(code) == "" {
		return signal.Skipped(emojiSentimentMeta, "empty file")
	}

	total := 0
	var parts []string

	for _, emoji := range aiHypeEmojis {
		if n := strings.Count(code, emoji); n > 0 {
			total += n
			parts = append(parts, fmt.Sprintf("%s (%d)", emoji, n))
		}
	}

	variety := len(parts)

	var score float64
	var summary string

	switch {
	case variety == 0:
		return signal.NewSignal(emojiSentimentMeta, 0.0)
	case variety >= 3 || total > 5:
		score, summary = 0.8, "High density of informal emojis in production code"
	default:
		score, summary = 0.3, "Informal emojis detected in comments or log messages"
	}

	return signal.NewSignal(emojiSentimentMeta, score, signal.Evidence{
		Detail: fmt.Sprintf("%s. Emojis found: %s", summary, strings.Join(parts, ", ")),
		Metrics: map[string]float64{
			"emoji_total":    float64(total),
			"emoji_distinct": float64(variety),
		},
	})
}
