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

	"github.com/dihedron/go-photo/files"
	"github.com/fatih/color"
	"github.com/mattn/go-sqlite3"
)

const DB_NAME string = "index.db"

func main() {

	dir := os.Args[1]

	duplicates := make(map[string][]string)

	db, statement, err := initDB(os.Args[1])
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer statement.Close()
	defer db.Close()

	err = filepath.Walk(dir, getFileWalker(db, statement, duplicates))
	if err != nil {
		log.Fatalf("error walking directory tree starting at %s: %v", os.Args[1], err)
	}
}

func initDB(path string) (*sql.DB, *sql.Stmt, error) {

	if ok, err := files.IsDir(path); !ok || err != nil {
		return nil, nil, fmt.Errorf("path %s is not an existing directory: %v", path, err)
	}

	dbpath := filepath.Join(path, DB_NAME)

	info, err := os.Stat(dbpath)
	var db *sql.DB
	if (err == nil && !info.IsDir()) || os.IsNotExist(err) {
		db, err = sql.Open("sqlite3", dbpath)
		if err != nil {
			return nil, nil, err
		}
		_, err = db.Exec(`create table if not exists files (hash text not null primary key, path text not null);`)
		if err != nil {
			log.Fatalf("error initialising table: %v\n", err)
		}
	}

	statement, err := db.Prepare("insert into files(hash, path) values(?, ?)")
	if err != nil {
		log.Fatalf("error creating pepared statement: %v", err)
	}

	return db, statement, err
}

func getFileWalker(db *sql.DB, statement *sql.Stmt, duplicates map[string][]string) func(string, os.FileInfo, error) error {
	hasher := sha256.New()
	red := color.New(color.FgRed).SprintFunc()
	return func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			//fmt.Printf("analysing %s...", path)

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
				switch err := err.(type) {
				case sqlite3.Error:
					if err.Code == 19 {
						fmt.Printf("%s", red(fmt.Sprintf("%-80s => %d: %s\n", path, err.Code, err.Error())))
					}
				default:
					fmt.Printf("unexpected error %T\n", err) // %T prints whatever type t has
				}
			} else {
				fmt.Printf("%-80s => %s\n", path, hash)
			}
		}
		return nil
	}
}
