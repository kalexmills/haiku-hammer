package db_test

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/kalexmills/haiku-enforcer/src/haikuhammer/db"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"path"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func TestMain(m *testing.M) {
	dbPath := fmt.Sprintf(path.Join("%s","test.db"), os.TempDir())

	// delete any existing database
	err := os.Truncate(dbPath, 0)

	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("could not truncate database file %s: %v", dbPath, err)
	}

	// open DB and load schema
	DB, err = sql.Open("sqlite3", dbPath)
	defer DB.Close()

	err = db.BootstrapDB(DB)
	if err != nil {
		log.Fatalf("could not open database %s: %v", dbPath, err)
	}

	m.Run()

	os.Remove(dbPath)
}

func TestHaikuDAO_Upsert(t *testing.T) {
	ctx := context.Background()

	rows, err := db.HaikuDAO.Upsert(ctx, DB, db.Haiku{1,1,1,"mention#3414", "not really a haiku"})

	assert.NoError(t, err)
	assert.EqualValues(t, 1, rows)

	haiku, err := db.HaikuDAO.FindByID(ctx, DB, 1)
	assert.NoError(t, err)
	assert.EqualValues(t, haiku.Content, "not really a haiku")
	assert.EqualValues(t, haiku.AuthorID, "mention#3414")

	db.HaikuDAO.Upsert(ctx, DB, db.Haiku{1,1,1,"changed_mention","updated haiku"})
	assert.NoError(t, err)
	assert.EqualValues(t, 1, rows)

	haiku, err = db.HaikuDAO.FindByID(ctx, DB, 1)
	assert.NoError(t, err)
	assert.EqualValues(t, haiku.Content, "updated haiku")
	assert.EqualValues(t, haiku.AuthorID, "mention#3414")
}

func TestHaikuDAO_Random(t *testing.T) {
	ctx := context.Background()

	db.HaikuDAO.Upsert(ctx, DB, db.Haiku{1,1,2,"mention#3414","not really a haiku"})
	db.HaikuDAO.Upsert(ctx, DB, db.Haiku{1,1,3,"mention#3414","also not haiku"})
	db.HaikuDAO.Upsert(ctx, DB, db.Haiku{1,1,4,"mention#3414","not even haiku"})

	// should not hit the below rows since filtering by guild_id
	db.HaikuDAO.Upsert(ctx, DB, db.Haiku{2,2,6,"mention#3414","not even haiku"})
	db.HaikuDAO.Upsert(ctx, DB, db.Haiku{2,2,7,"mention#3414","not even haiku"})
	db.HaikuDAO.Upsert(ctx, DB, db.Haiku{2,2,8,"mention#3414","not even haiku"})

	for i := 0; i < 10; i++ {
		result, err := db.HaikuDAO.Random(ctx, DB, "1")
		assert.NoError(t, err)
		assert.True(t, 1 <= result.MessageID && result.MessageID <= 4)
		assert.Equal(t, 1, result.GuildID)
		assert.Equal(t, 1, result.ChannelID)
	}

	result, err := db.HaikuDAO.Random(ctx, DB, "123") // should be empty
	assert.NoError(t, err)
	assert.Empty(t, result.Content)
}