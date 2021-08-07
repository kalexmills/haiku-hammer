package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/kalexmills/haiku-enforcer/src/haikuhammer"
	"github.com/kalexmills/haiku-enforcer/src/haikuhammer/db"
	"io"
	"log"
	"os"
	"strings"
)

const guildID = 690680416373571585
const channelID = 704842231227482182

func main() {
	f, err := os.Open("scripts/wikipedia-enhance/data/gen-chat.csv.txt")
	FatalError(err)
	defer f.Close()

	s := csv.NewReader(f)

	DB, err := sql.Open("sqlite3", "scripts/haiku-extract/haikuDB.sqlite3")
	if err != nil {
		log.Fatalf("cannot open database from: %v", err)
	}

	ctx := context.Background()
	messageID := 0
	for  {
		records, err := s.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		content := strings.TrimSpace(records[3])
		if err := haikuhammer.IsHaiku(content); err == nil {
			_, err := db.HaikuDAO.Upsert(ctx, DB, db.Haiku{
				ChannelID: channelID,
				GuildID: guildID,
				MessageID: messageID,
				AuthorID: records[0],
				Content: content,
			})
			if err != nil {
				log.Fatal("couldn't write to db", err)
			}
			messageID++
		}
	}
}

func lineParser(s string) string {
	tokens := strings.Split(s, ",")
	if len(tokens) < 4 {
		return ""
	}
	return strings.Trim(tokens[3], " \"")
}

func FatalError(err error) {
	if err != nil {
		fmt.Printf("encountered error: %v\n", err)
		os.Exit(1)
	}
}