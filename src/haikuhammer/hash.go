package haikuhammer

import (
	"context"
	"crypto/md5"
	"database/sql"
	"github.com/kalexmills/haiku-enforcer/src/haikuhammer/db"
	"log"
	"strings"
)

func DuplicateHash(haiku string) [md5.Size]byte {
	s := strings.ToUpper(hashStrip(haiku))
	sum := md5.New()
	sum.Write([]byte(s))

	out := make([]byte, 0, md5.Size)
	out = sum.Sum(out[:])

	var result [md5.Size]byte
	for i := 0; i < md5.Size; i++ {
		result[i] = out[i]
	}
	return result
}

func hashStrip(s string) string {
	return stripBytes(s, func(b byte) bool {
		return ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z') || b == ' ' || b == '\n'
	})
}

// UpdateHashes ensures all haiku have their hashes loaded into the table. It's intended
// to be run on a separate thread on startup.
func UpdateHashes(sqlDB *sql.DB) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("recovered from panic in UpdateHashes: %v", err)
			return
		}
	}()
	log.Println("beginning UpdateHashes.")
	ctx := context.Background()
	rows, err := sqlDB.QueryContext(ctx, `SELECT message_id, content FROM haiku`)
	if err == sql.ErrNoRows {
		return
	}
	if err != nil {
		log.Println("encountered error while updating hashes,", err)
	}
	defer rows.Close()
	var (
		messageID int
		content string
	)
	insertCount := 0
	for rows.Next() {
		err = rows.Scan(&messageID, &content)
		if err != nil {
			log.Println("encountered error while scanning hashes,", err)
			return
		}
		hash := DuplicateHash(content)
		count, _ := db.HaikuHashDAO.Upsert(ctx, sqlDB, messageID, hash[:])
		if count != 0 {
			insertCount++
		}
	}
	log.Printf("upserted %d new haiku hashes", insertCount)
}