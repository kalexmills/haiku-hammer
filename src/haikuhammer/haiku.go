package haikuhammer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrThreeLines = errors.New("This doesn't seem to me like a traditional haiku; it doesn't have three lines.")
)

// IsHaiku returns nil if the provided string is a properly formed 3-line haiku with
// 5/7/5 structure, otherwise it returns an error explaining any issues it has found.
func IsHaiku(str string) error {
	trimmed := strings.Trim(str, " \n\t")
	cleaned := cleanEmoji(trimmed)
	lines := strings.Split(cleaned, "\n")
	if len(lines) != 3 {
		return ErrThreeLines
	}
	line1, err1 := lineSyllableCount(lines[0])
	line2, err2 := lineSyllableCount(lines[1])
	line3, err3 := lineSyllableCount(lines[2])
	return evaluateHaiku([3]int{line1,line2,line3}, []error{err1, err2, err3})
}

// evaluateHaiku returns nil if the provided data represents a Haiku, otherwise it explains why the provided string is
// not a haiku
func evaluateHaiku(lines [3]int, errs []error) error {
	if lines != [3]int{5,7,5} {
		errs = append(errs, fmt.Errorf("I counted a syllable structure of %d/%d/%d, but I expected 5/7/5", lines[0], lines[1], lines[2]))
	}
	errStr := "Hmmm, this doesn't seem like a traditional English Haiku; here's why:"
	var hasErr bool
	for _, err := range errs {
		if err == nil {
			continue
		}
		hasErr = true
		errStr += "\n- " + err.Error()
	}
	if hasErr {
		return errors.New(errStr)
	}
	return nil
}

var EmojiRegex *regexp.Regexp
func cleanEmoji(s string) string {
	return strings.TrimSpace(EmojiRegex.ReplaceAllString(s, ""))
}

func lineSyllableCount(line string) (int, error) {
	words := strings.Split(line, " ")
	count := 0

	var unknownWords []string
	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		syllables, ok := CountSyllables(word)
		if !ok {
			unknownWords = append(unknownWords, word)

		}
		count += syllables
	}
	if len(unknownWords) != 0 {
		return 0, fmt.Errorf("I don't know the words: %s", strings.Join(unknownWords, ", "))
	}
	return count, nil
}