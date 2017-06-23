package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

func main() {

	dir := os.Args[1]

	files := make(map[string][]string)

	if err := filepath.Walk(dir, getFileWalker(files)); err != nil {
		log.Fatalf("error walking directory tree starting at %s: %v", os.Args[1], err)
	}
}

func getFileWalker(files map[string][]string) func(string, os.FileInfo, error) error {
	hasher := sha256.New()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	return func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			stream, err := os.Open(path)
			if err != nil {
				fmt.Printf("%s", yellow(fmt.Sprintf("%-80s => %v\n", path, err)))
				return nil
			}
			defer stream.Close()
			hasher.Reset()
			if _, err := io.Copy(hasher, stream); err != nil {
				log.Fatal(err)
			}
			hash := hex.EncodeToString(hasher.Sum(nil))
			if duplicates, ok := files[hash]; ok {
				fmt.Printf("%s", red(fmt.Sprintf("%-80s => %v\n", path, duplicates)))
				files[hash] = append(files[hash], path)
			} else {
				//fmt.Printf("%-80s => %s\n", path, hash)
				files[hash] = []string{path}
			}
		}
		return nil
	}
}
