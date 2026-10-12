package signals

import (
	"fmt"
	"sentinel/pkg/models"
	"strings"
)

// Scans for clusters of emojis in comments and logger messages,
// which are an uncommon pattern in production codebases.
func CheckEmojiSentiment(code string) models.Signal {
	aiHypeEmojis := []string{"🚀", "✨", "✅", "💡", "🛠️", "🤖"}

	emojiMatches := 0
	found := make(map[string]int)

	for _, emoji := range aiHypeEmojis {
		count := strings.Count(code, emoji)
		if count > 0 {
			emojiMatches += count
			found[emoji] = count
		}
	}

	variety := len(found)
	var score float64
	var description string

	if variety >= 3 || emojiMatches > 5 {
		score = 0.8
		description = "High density of informal emojis in production code"
	} else if variety >= 1 {
		score = 0.3
		description = "Informal emojis detected in comments or log messages"
	} else {
		score = 0.0
		description = "No typical emoji patterns found"
	}

	evidenceParts := []string{}
	for e, c := range found {
		evidenceParts = append(evidenceParts, fmt.Sprintf("%s (%d)", e, c))
	}

	return models.Signal{
		Name:        "emoji_sentiment",
		Score:       score,
		Description: description,
		Evidence:    FormatEvidence("Emojis found: %s", strings.Join(evidenceParts, ", ")),
	}
}
