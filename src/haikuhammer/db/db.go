package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
)

//go:embed scripts/*.sql
var bootstrapScripts embed.FS

// BootstrapDB attempts to execute all files ending in .sql from the provided directory against the
// provided database, in alphabetical order by filename. If no files are found, an error is returned.
func BootstrapDB(DB *sql.DB) error {
	foundSQLFile := false
	scripts, err := bootstrapScripts.ReadDir("scripts")
	if err != nil {
		return err
	}
	for _, finfo := range scripts {
		if finfo.IsDir() {
			continue
		}
		foundSQLFile = true

		script, err := bootstrapScripts.ReadFile("scripts/"+finfo.Name())
		if err != nil {
			return err
		}
		_, err = DB.Exec(string(script))
		if err != nil {
			log.Printf("could not execute bootstrap script %s: %v", finfo.Name(), err)
			return err
		}
		log.Printf("executed bootstrap script %s", finfo.Name())
	}
	if !foundSQLFile {
		return fmt.Errorf("could not find any *.sql files in schema folder scripts")
	}
	return nil
}