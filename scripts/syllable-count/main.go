package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
)

const Filename = "data/cmudict-0.7b.txt"

func main() {
	f, err := readFile()
	if err != nil {
		fmt.Printf("encountered error: %v\n", err)
		os.Exit(1)
	}
	entries := parseFile(f)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].word < entries[j].word
	})

	for _, entry := range entries {
		fmt.Print(entry.word)
		for _, count := range entry.syllableCounts {
			fmt.Printf(" %d", count)
		}
		fmt.Println()
	}
	os.Exit(0)
}

func readFile() ([]byte, error) {
	f, err := ioutil.ReadFile(Filename)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func parseFile(file []byte) []Entry {
	countsByWord := make(map[string][]int)

	// find initial counts
	lines := bytes.Split(file, []byte("\n")) // not the fastest.
	for _, line := range lines {
		word, count, ok := parseLine(line)
		if ok {
			countsByWord[word] = append(countsByWord[word], count)
		}
	}

	// deduplicate counts (for multiple pronounciations)
	toChange := make(map[string][]int)
	for word, counts := range countsByWord {
		if len(counts) == 1 {
			continue
		}
		seen := make(map[int]struct{})
		for _, count := range counts {
			seen[count] = struct{}{}
		}
		if len(seen) != len(counts) {
			for count := range seen {
				toChange[word] = append(toChange[word], count)
			}
		}
	}
	for word, counts := range toChange {
		countsByWord[word] = counts
	}

	// return results as slice of Entry
	var result []Entry
	for word, counts := range countsByWord {
		result = append(result, Entry{word, counts})
	}
	return result
}

func parseLine(line []byte) (string, int, bool) {
	if bytes.HasPrefix(line, []byte(";;;")) { // comment
		return "", 0, false
	}
	tokens := bytes.Split(line, []byte("  "))
	if len(tokens) != 2 {
		return "", 0, false
	}
	word := string(tokens[0])
	if word[len(word)-1] == ')' { // remove extra pronounciation count
		word = word[:len(word)-3]
	}
	count := countSyllables(tokens[1])
	return word, count, true
}

func countSyllables(syllable []byte) int {
	phonemes := bytes.Split(syllable, []byte(" "))
	vowelCount := 0
	for _, phoneme := range phonemes {
		if len(phoneme) < 2 {
			continue
		}
		if _, ok := Vowels[string(phoneme[:2])]; ok {
			vowelCount++
		}
	}
	return vowelCount
}

var Vowels map[string]struct{}

func init() {
	Vowels = make(map[string]struct{})
	vowels := []string{"AA", "AE", "AH", "AO", "AW", "AY", "EH", "ER", "EY", "IH", "IY", "OW", "OY", "UH", "UW"}
	for _, vowel := range vowels {
		Vowels[vowel] = struct{}{}
	}
}

type Entry struct {
	word string
	syllableCounts []int
}


