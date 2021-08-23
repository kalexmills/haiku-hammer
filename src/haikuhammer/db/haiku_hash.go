package db

import (
	"context"
	"fmt"
	"github.com/jonbodner/proteus"
	"log"
)

var HaikuHashDAO HaikuHashDaoImpl

type HaikuHashDaoImpl struct {
	Upsert    func(ctx context.Context, e proteus.ContextExecutor, mid int, md5Sum []byte) (int64, error) `proq:"q:upsert" prop:"mid,md5Sum"`
	FindByMD5 func(ctx context.Context, e proteus.ContextQuerier, md5Sum []byte) (int64, error)           `proq:"q:findByMD5" prop:"md5Sum"`
}

func init() {
	m := proteus.MapMapper{
		"upsert":    `INSERT INTO haiku_hash (message_id, md5_sum) VALUES (:mid:, :md5Sum:)
  				      ON CONFLICT (message_id) 
					  DO UPDATE SET md5_sum = excluded.md5_sum`,
		"findByMD5": `SELECT message_id FROM haiku_hash WHERE md5_sum = :md5Sum:`,
	}
	err := proteus.ShouldBuild(context.Background(), &HaikuHashDAO, proteus.Sqlite, m)
	if err != nil {
		panic(err)
	}
}

func CheckHash(ctx context.Context, e proteus.ContextWrapper, mid int, hash [16]byte) error {
	midFound, err := HaikuHashDAO.FindByMD5(ctx, e, hash[:])
	if err != nil {
		log.Println("haiku was found to be plagiarized; original message_id:", midFound, ",", err)
		return err
	}
	_, err = HaikuHashDAO.Upsert(ctx, e, mid, hash[:])
	if err != nil {
		log.Println("could not store haiku hash in database,", err)
		return fmt.Errorf("error while storing haiku hash: %w", err)
	}
	return nil
}