package haikuhammer

import (
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func CountSyllables(word string) (int, bool) {
	cleaned := cleanWord(word)
	counts, ok := SyllableCounts[cleaned]
	if ok {
		return counts[0], true
	}
	count, ok := countAbbreviation(word)
	if ok {
		return count, true
	}
	count, ok = countCompound(cleaned)
	if ok {
		return count, true
	}
	return 0, false
}

func countCompound(word string) (int, bool) {
	if word == "" {
		return 0, true
	}
	curr := DictionaryTrie
	best := 1000
	for i := 0; i < len(word); i++ {
		curr = curr.Child(word[i])
		if curr == nil {
			break
		}
		if curr.isWord {
			count := SyllableCounts[word[:i+1]]
			rest, ok := countCompound(word[i+1:])
			if ok {
				if best > count[0] + rest {
					best = count[0] + rest
				}
			}
		}
	}
	if best == 1000 {
		return 0, false
	}
	return best, true
}

func countAbbreviation(word string) (int, bool) {
	if !isAbbreviation(word) {
		return 0, false
	}
	count := 0
	for _, c := range word {
		if 'A' <= c && c <= 'Z' {
			count++ // not all letters are the same
		}
		if c == 'W' {
			count += 2
		}
	}
	return count, true
}

func isAbbreviation(word string) bool {
	return AbbrevRegex.MatchString(word)
}

func cleanWord(s string) string {
	replaced := strings.NewReplacer("’","'","‘","'").Replace(s)
	return strip(strings.ToUpper(replaced))
}

func strip(s string) string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') || b == '\''{
			result.WriteByte(b)
		}
	}
	return result.String()
}

var AbbrevRegex *regexp.Regexp

//go:embed data/english-syllables.txt
var syllablesFile string

var SyllableCounts map[string][]int

var DictionaryTrie *TrieNode

func init() {
	initDictionary()
	initAbbrevRegex()
}

func initDictionary() {
	SyllableCounts = make(map[string][]int)
	DictionaryTrie = &TrieNode{}

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
		SyllableCounts[word] = counts

		DictionaryTrie.Insert(word)
	}
}

func initAbbrevRegex() {
	var err error
	AbbrevRegex, err = regexp.Compile("^[A-Z\\.]+$")
	if err != nil {
		panic(fmt.Errorf("could not parse regex: %w", err))
	}
}

type TrieNode struct {
	isWord bool
	children [26]*TrieNode
}

func (n *TrieNode) Insert(word string) {
	if len(word) == 0 {
		n.isWord = true
		return
	}

	idx := word[0] - 'A'
	if idx < 0 || idx > 26 {
		return
	}

	if child := n.children[idx]; child == nil {
		n.children[idx] = &TrieNode{}
	}
	n.children[idx].Insert(word[1:])
}

func (n *TrieNode) HasPrefix(str string) bool {
	if n == nil {
		return false
	}
	if len(str) == 0 {
		return n.isWord
	}

	return n.Child(str[0]).HasPrefix(str[1:])
}

func (n *TrieNode) Child(ch byte) *TrieNode {
	idx := ch - 'A'
	if idx < 0 || idx > 26 {
		return nil
	}
	return n.children[idx]
}