package db

import (
	"context"
	"github.com/jonbodner/proteus"
)

type Haiku struct {
	GuildID       int    `prof:"guild_id"`
	ChannelID     int    `prof:"channel_id"`
	MessageID     int    `prof:"message_id"`
	AuthorMention string `prof:"author_mention"`
	Content       string `prof:"content"`
}

var HaikuDAO HaikuDaoImpl

type HaikuDaoImpl struct {
	Upsert func(ctx context.Context, e proteus.ContextExecutor, h Haiku) (int64, error) `proq:"q:upsert" prop:"h"`
	Random func(ctx context.Context, e proteus.ContextQuerier, guildID string) (Haiku, error)           `proq:"q:random" prop:"guildID"`
	// FindByID is only intended for testing
	FindByID func(ctx context.Context, e proteus.ContextQuerier, messageID int) (Haiku, error)           `proq:"q:findByID" prop:"messageID"`
}

func init() {
	m := proteus.MapMapper{
		"upsert": `INSERT INTO haiku (guild_id, channel_id, message_id, author_mention, content)
				   VALUES (:h.GuildID:,:h.ChannelID:,:h.MessageID:,:h.AuthorMention:,:h.Content:)
                   ON CONFLICT(guild_id, channel_id, message_id)
				   DO UPDATE SET content = excluded.content`,
		"findByID": `SELECT * FROM haiku WHERE message_id = :messageID:`,
	    "random": `SELECT * FROM haiku WHERE guild_id = :guildID: ORDER BY RANDOM() LIMIT 1`,
	}
	err := proteus.ShouldBuild(context.Background(), &HaikuDAO, proteus.Sqlite, m)
	if err != nil {
		panic(err)
	}
}