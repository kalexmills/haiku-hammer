package dict

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

//go:embed data/english-syllables.txt
var syllablesFile string

var syllableCounts map[string][]int

func SyllableCounts(word string) ([]int, bool) {
	result, ok := syllableCounts[word]
	return result, ok
}

func IsWord(word string) bool {
	_, ok := syllableCounts[word]
	return ok
}

var TrieRoot *TrieNode

func init() {
	syllableCounts = make(map[string][]int)
	TrieRoot = &TrieNode{}

	lines := strings.Split(syllablesFile, "\n")
	for lineNum, line := range lines {
		tokens := strings.Split(line, " ")
		word := tokens[0]
		var counts []int
		for _, token := range tokens[1:] {
			count, err := strconv.Atoi(token)
			if err != nil {
				panic(fmt.Errorf("could not parse line %d: %w", lineNum, err))
			}
			counts = append(counts, count)
		}
		if len(counts) == 0 {
			continue
		}

		syllableCounts[word] = counts

		TrieRoot.insert(word)
	}
}