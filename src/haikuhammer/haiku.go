package haikuhammer

import (
	"fmt"
	"regexp"
	"strings"
)

func IsHaiku(str string) bool {
	trimmed := strings.Trim(str, " \n\t")
	cleaned := cleanEmoji(trimmed)
	lines := strings.Split(cleaned, "\n")
	if len(lines) != 3 {
		return false
	}
	line1, err := lineSyllableCount(lines[0])
	if err != nil {
		return false
	}
	line2, err := lineSyllableCount(lines[1])
	if err != nil {
		return false
	}
	line3, err := lineSyllableCount(lines[2])
	if err != nil {
		return false
	}
	return line1 == 5 && line2 == 7 && line3 == 5
}

var EmojiRegex *regexp.Regexp
func cleanEmoji(s string) string {
	return strings.TrimSpace(EmojiRegex.ReplaceAllString(s, ""))
}

func lineSyllableCount(line string) (int, error) {
	words := strings.Split(line, " ")
	count := 0
	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		syllables, ok := CountSyllables(word)
		if !ok {
			return 0, fmt.Errorf("unknown word %s", word)
		}
		count += syllables
	}
	return count, nil
}