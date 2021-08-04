package main

import (
	"bufio"
	"fmt"
	"github.com/kalexmills/haiku-enforcer/src/dict"
	"os"
	"sort"
	"strings"
)

const Filename = "data/wikipedia.talkpages.conversations.txt"

type Datasource struct {
	filename string
	lineParser func(string) string
}

var Unescaper = strings.NewReplacer("\\/","/", "\\\"", "\"", "''''","'", "''","'")

var sources = map[string]Datasource{
	"wikipedia": {
		filename: "data/wikipedia.talkpages.conversations.txt",
		lineParser: func(s string) string {
			tokens := strings.Split(s, "+++$+++")
			if len(tokens) < 8 {
				return ""
			}
			cleaned := strings.TrimSpace(tokens[7]) // 7th index is the 'cleaned' content
			return Unescaper.Replace(cleaned)
		},
	},
	"gen-chat": {
		filename: "data/gen-chat.csv.txt",
		lineParser: func(s string) string {
			tokens := strings.Split(s, ",")
			if len(tokens) < 4 {
				return ""
			}
			return strings.Trim(tokens[3], " \"")
		},
	},
}

func main() {
	source := sources["gen-chat"]

	f, err := os.Open(source.filename)
	FatalError(err)
	defer f.Close()

	counts := make(map[string]int)
	s := bufio.NewScanner(f)

	for s.Scan() {
		str := strings.TrimSpace(s.Text())
		if str == "" {
			continue
		}
		tokens := tokenize(source.lineParser(str))
		for _, t := range tokens {
			cleaned := clean(t)
			if cleaned == "" || dict.IsWord(cleaned) {
				continue
			}
			counts[cleaned]++
		}
	}

	type result struct {
		str string
		count int
	}
	var results []result
	for s, count := range counts {
		if count == 1 {
			continue // we don't care about one-offs.
		}
		results = append(results, result{s, count})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].count > results[j].count
	})
	for _, result := range results {
		fmt.Println(result.str, result.count)
	}
}


func tokenize(s string) []string {
	return strings.Split(s, " ")
}

var Requoter = strings.NewReplacer("’","'","‘","'")

func clean(s string) string {
	if strings.HasPrefix(s, "[") ||
		strings.HasPrefix(s, "<") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://"){
		return ""
	}
	replaced := Requoter.Replace(s)
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
	return strings.Trim(result.String(), "'")
}

func FatalError(err error) {
	if err != nil {
		fmt.Printf("encountered error: %v\n", err)
		os.Exit(1)
	}
}