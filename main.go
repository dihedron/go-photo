package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

	dir := os.Args[1]

	db, err := initDB(os.Args[1])
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer db.Close()

	err = filepath.Walk(dir, getFileWalker(db))
	if err != nil {
		log.Fatalf("error walking directory tree starting at %s: %v", os.Args[1], err)
	}
}

func initDB(path string) (*sql.DB, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not an existing directory", path)
	}
	dbpath := filepath.Join(path, ".index.db")
	info, err = os.Stat(dbpath)
	var db *sql.DB
	if (err == nil && !info.IsDir()) || os.IsNotExist(err) {
		db, err = sql.Open("sqlite3", dbpath)
		if err != nil {
			return nil, err
		}
		_, err = db.Exec(`create table if not exists files (hash text not null primary key, path text not null);`)
		if err != nil {
			log.Fatalf("error initialising table: %v\n", err)
		}
	}
	return db, err
}

func getFileWalker(db *sql.DB) func(string, os.FileInfo, error) error {
	hasher := sha256.New()
	statement, err := db.Prepare("insert into files(hash, path) values(?, ?)")
	if err != nil {
		log.Fatalf("error creating pepared statement: %v", err)
	}
	return func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fmt.Printf("analysing %s...", path)
			stream, err := os.Open(path)
			if err != nil {
				log.Fatal(err)
			}
			defer stream.Close()
			hasher.Reset()
			if _, err := io.Copy(hasher, stream); err != nil {
				log.Fatal(err)
			}
			hash := hex.EncodeToString(hasher.Sum(nil))
			_, err = statement.Exec(hash, path)
			if err != nil {
				fmt.Printf(" error %v\n", err)
			}
			fmt.Printf(" hash is %s\n", hash)
		} else {
			//fmt.Printf(" it's a directory: skip!\n")
		}
		return nil
	}
}
