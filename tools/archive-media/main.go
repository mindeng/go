package main

import (
	"fmt"
	"log"
	"os"
	"path"
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

type ArchiveResultType int

const (
	Archived ArchiveResultType = iota
	IgoreExisted
	IgnoreErrorTime
)

type ArchiveResult struct {
	src    string
	dst    string
	result ArchiveResultType
	err    error
}
type MediaInfo struct {
	path    string
	created time.Time
	err     error
}

func archiveFiles(mediaFiles <-chan MediaInfo, dstDir string, results chan<- ArchiveResult) {
	for mediaFile := range mediaFiles {
		src := mediaFile.path
		// log.Println("got ", src)
		created := mediaFile.created

		if created.Year() < 1980 {
			// fmt.Fprintf(os.Stderr, "time error: %v %s\n", created, path)
			results <- ArchiveResult{src, "", IgnoreErrorTime, nil}
			continue
		}

		dst := path.Join(dstDir, fmt.Sprintf("%02d/%02d/%02d", created.Year(), created.Month(), created.Day()), path.Base(src))
		dstDir := path.Dir(dst)
		if _, err := os.Stat(dstDir); os.IsNotExist(err) {
			os.MkdirAll(dstDir, 0755)
		}

		if info, err := os.Stat(dst); err == nil {
			t, _ := minlib.FileOriginalTime(dst)
			if t == created {
				if srcInfo, err := os.Stat(src); err == nil && srcInfo.Size() == info.Size() {
					results <- ArchiveResult{src, dst, IgoreExisted, nil}
					continue
				}
			}
		}

		err := minlib.CopyFile(dst, mediaFile.path)
		results <- ArchiveResult{mediaFile.path, dst, Archived, err}
	}
	close(results)
}

func walkDirectory(dir string, out chan MediaInfo) {
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
			created, err := minlib.FileOriginalTime(path)
			out <- MediaInfo{path, created, err}
			return nil
		default:
			return nil
		}
	})

	// log.Println("q closed")
	close(out)
}

func archive(src string, dst string) {
	mediaFiles := make(chan MediaInfo, 100)
	results := make(chan ArchiveResult, 1000)
	go walkDirectory(src, mediaFiles)
	go archiveFiles(mediaFiles, dst, results)

	start := time.Now()

	func() {
		var errorTimeFiles []string
		var existedFiles []string
		for result := range results {
			if result.err != nil {
				fmt.Fprintf(os.Stderr, "archive failed: %v %s\n", result.err, result.src)
			} else {
				switch result.result {
				case Archived:
					fmt.Printf("%s -> %s\n", result.src, result.dst)
				case IgnoreErrorTime:
					errorTimeFiles = append(errorTimeFiles, result.src)
				case IgoreExisted:
					existedFiles = append(existedFiles, result.src)
				}
				// fmt.Printf("%v %s archived\n", result.created, result.path)
				// path, err := result.path, result.err
			}
		}

		fmt.Fprintf(os.Stderr, "%s", "Files with error time: \n")
		for _, path := range errorTimeFiles {
			fmt.Fprintf(os.Stderr, "%s\n", path)
		}

		fmt.Fprintf(os.Stderr, "%s", "Files existed: \n")
		for _, path := range existedFiles {
			fmt.Fprintf(os.Stderr, "%s\n", path)
		}
	}()

	elapsed := time.Since(start)
	log.Printf("took %s", elapsed)
}

func main() {
	archive(os.Args[1], os.Args[2])
}
