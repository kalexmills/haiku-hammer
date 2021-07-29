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
	count, ok := countWord(cleaned)
	if ok {
		return count, true
	}
	count, ok = countAbbreviation(word)
	if ok {
		return count, true
	}
	count, ok = countCompound(cleaned)
	if ok {
		return count, true
	}
	return 0, false
}

func countWord(cleaned string) (int, bool) {
	counts, ok := SyllableCounts[cleaned]
	if ok && len(counts) > 0 {
		return counts[0], true
	}
	n := len(cleaned)
	if cleaned[n-1] == 'Y' {
		counts, ok = SyllableCounts[cleaned[:n-1]]
		if ok && len(counts) > 0 {
			return counts[0] + 1, true
		}
	}
	if cleaned[n-1] == 'S' {
		counts, ok = SyllableCounts[cleaned[:n-1]]
		if ok && len(counts) > 0 {
			return counts[0], true
		}
	}
	return 0, false
}

func countCompound(cleaned string) (int, bool) {
	if len(cleaned) > 1000 {
		return 0, false // we're just not allowing compound words this long (preventing DoS), sorry!
	}
	// recursively crawls a trie, looking for valid break points in the word.
	// every possible segmentation of cleaned is tested, using the trie helps to end the search early
	// in case no other words exist, and helps to ensure only valid breakpoints are recursively checked.
	if cleaned == "" {
		return 0, true
	}
	curr := DictionaryTrie // all words start from the beginning
	best := 1000
	for i := 0; i < len(cleaned); i++ {
		curr = curr.Child(cleaned[i]) // test the next letter against the next character in the trie
		if curr == nil {
			break
		}
		if curr.isWord { // found a prefix that's a word. Count its syllables and start over with the remainder
			count := SyllableCounts[cleaned[:i+1]]
			rest, ok := countCompound(cleaned[i+1:])
			if ok {
				// we were able to complete the suffix, add its count to the prefix and check to see
				// if it's the best breakdown so far. But keep going to test all the other prefixes in
				// case we have better options.
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
		if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' {
			count++
		}
		if c == 'W' || c == 'w' {
			count += 2 // W is 3 syllables; 2 more than the 1 we added above
		}
	}
	return count, true
}

func isAbbreviation(word string) bool {
	trimmed := strings.TrimFunc(word, func(r rune) bool {
		return !('A' <= r && r <= 'Z') && !('a' <= r && r <= 'z')
	})
	return AbbrevRegex.MatchString(trimmed)
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
	initDictionaryAndCounts()
	initAbbrevRegex()
	initEmojiRegex()
}

func initDictionaryAndCounts() {
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
		if len(counts) == 0 {
			continue
		}

		SyllableCounts[word] = counts

		DictionaryTrie.Insert(word)
	}
}

func initAbbrevRegex() {
	var err error
	AbbrevRegex, err = regexp.Compile("^([A-Z\\.]+|[a-z])$")
	if err != nil {
		panic(fmt.Errorf("could not parse regex: %w", err))
	}
}

func initEmojiRegex() {
	var err error
	EmojiRegex, err = regexp.Compile("\\:.+\\:")
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