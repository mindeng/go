package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mindeng/go/minlib"
)

type ArchiveCallback func(path string, err error) error

type ArchiveFunc func(path string, created time.Time, info os.FileInfo) error

type ErrArchiveIgnore struct {
	info string
}

func (err ErrArchiveIgnore) Error() string {
	return err.info
}

type ArchiveResult struct {
	path    string
	created time.Time
	err     error
}

func archiveFiles(q <-chan string) <-chan ArchiveResult {
	chanList := make([]<-chan ArchiveResult, 4)
	done := make(chan bool)
	for i := 0; i < len(chanList); i++ {
		c := make(chan ArchiveResult, 100)
		chanList[i] = c

		go func(c chan ArchiveResult) {

			for p := range q {
				// log.Println("got ", p)
				created, err := minlib.FileOriginalTime(p)
				c <- ArchiveResult{p, created, err}
			}
			// log.Println("close ", c)
			close(c)
		}(c)
	}

	results := make(chan ArchiveResult, 100)
	for _, ch := range chanList {
		go func(c <-chan ArchiveResult) {
			for result := range c {
				results <- result
			}
			done <- true
		}(ch)
	}

	go func() {
		for i := 0; i < len(chanList); i++ {
			<-done
		}
		close(results)
	}()

	return results
}

func walkDirectory(dir string, q chan string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "walk error: %v %s\n", err, path)
			return nil
		}
		// Ignore directory
		if info.IsDir() {
			return nil
		}
		// Ignore invalid files
		if info.Name()[0] == '.' && info.Size() == 4096 {
			return nil
		}
		switch strings.ToLower(filepath.Ext(info.Name())) {
		case ".jpg", ".png", ".arw", ".nef", ".avi", ".mp4", ".mov", ".m4v":
			// need to archive
			// log.Println("put ", path)
			q <- path
			return nil
		default:
			return nil
		}
	})

	// log.Println("q closed")
	close(q)
}

func archiveDirectory(src string) {
	q := make(chan string, 100)
	results := archiveFiles(q)

	go walkDirectory(src, q)

	start := time.Now()

	func() {
		for result := range results {
			if result.err != nil {
				fmt.Fprintf(os.Stderr, "failed %v %s\n", result.err, result.path)
			} else {
				fmt.Printf("%v %s archived\n", result.created, result.path)
				created, path := result.created, result.path
				if created.Year() < 1980 {
					fmt.Fprintf(os.Stderr, "time error: %v %s\n", created, path)
				} else {
					fmt.Printf("archive %v %s\n", created, path)
				}
			}
		}
	}()

	elapsed := time.Since(start)
	log.Printf("took %s", elapsed)
}

func main() {
	archiveDirectory(os.Args[1])
}
