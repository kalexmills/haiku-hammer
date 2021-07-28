package haiku_enforcer

import (
	"fmt"
	"strings"
)

func IsHaiku(str string) bool {
	trimmed := strings.Trim(str, " \n\t")
	lines := strings.Split(trimmed, "\n")
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

func isVowel(chr_ rune) bool {
	chr := string(chr_)
	if (strings.EqualFold(chr, "a") || strings.EqualFold(chr, "e") ||
		strings.EqualFold(chr, "i") ||
		strings.EqualFold(chr, "o") ||
		strings.EqualFold(chr, "u") ||
		strings.EqualFold(chr, "y")) {
		return true
	}
	return false
}