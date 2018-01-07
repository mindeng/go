package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
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
	CopyFailed
	CopyConflict
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

type CopyFileTask struct {
	src string
	dst string
}

func startCopyFileService(tasks <-chan CopyFileTask, results chan<- ArchiveResult) *sync.WaitGroup {
	var wg sync.WaitGroup
	const concurrentNum = 2
	wg.Add(concurrentNum)

	copy := func() {
		defer wg.Done()
		for t := range tasks {
			var err error
			if moveFlag {
				err = os.Rename(t.src, t.dst)
			} else {
				err = minlib.CopyFile(t.dst, t.src)
			}
			result := Archived
			if err != nil {
				result = CopyFailed
			}
			results <- ArchiveResult{t.src, t.dst, result, err}
		}
	}

	for i := 0; i < concurrentNum; i++ {
		go copy()
	}

	return &wg
}

func archiveFiles(mediaFiles <-chan MediaInfo, dstDir string, results chan<- ArchiveResult) {
	tasks := make(chan CopyFileTask, 100)
	wg := startCopyFileService(tasks, results)

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
			if err = os.MkdirAll(dstDir, 0755); err != nil {
				results <- ArchiveResult{src, dst, CopyFailed, err}
				break
			}
		}

		if info, err := os.Stat(dst); err == nil {
			t, _ := FileTime(dst)
			if t == created {
				if srcInfo, err := os.Stat(src); err == nil && srcInfo.Size() == info.Size() {
					if minlib.EqualFile(src, dst) {
						results <- ArchiveResult{src, dst, IgoreExisted, nil}
						continue
					}
				}
			}
			results <- ArchiveResult{src, dst, CopyConflict, errors.New("file conflicted")}
			continue
		}

		// err := minlib.CopyFile(dst, mediaFile.path)
		// results <- ArchiveResult{mediaFile.path, dst, Archived, err}
		tasks <- CopyFileTask{src, dst}
	}
	close(tasks)

	wg.Wait()

	close(results)
}

func FileTime(path string) (time.Time, error) {
	created, err := minlib.FileOriginalTime(path)

	if created.Year() >= 1980 && created.Year() <= 2100 {
		return created, nil
	} else {
		log.Fatalf("Error created time: %s %v\n", path, created)
	}

	if err != nil {
		if fi, err := os.Stat(path); err == nil {
			return fi.ModTime(), nil
		}
	}
	return created, err
}

func walkDirectory(dir string, out chan MediaInfo) {
	done := make(chan bool, concurrentNum)
	tasks := make(chan string, 100)

	extractTime := func() {
		for path := range tasks {
			created, err := FileTime(path)
			out <- MediaInfo{path, created, err}
		}
		done <- true
	}

	for i := 0; i < concurrentNum; i++ {
		go extractTime()
	}

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
		case ".jpg", ".jpeg", ".png", ".arw", ".nef", ".avi", ".mp4", ".mov", ".m4v", ".m4a":
			// need to archive
			// log.Println("put ", path)
			tasks <- path
			return nil
		default:
			return nil
		}
	})

	close(tasks)

	for i := 0; i < concurrentNum; i++ {
		<-done
	}

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
		var conflictedFiles []string
		var copyFailed []string

		copiedNum, duplicated := 0, 0
		for result := range results {
			if result.err != nil {
				fmt.Fprintf(os.Stderr, "[error] %v: %s %s\n", result.err, result.src, result.dst)
			}

			switch result.result {
			case CopyFailed:
				copyFailed = append(copyFailed, result.src)
			case CopyConflict:
				conflictedFiles = append(conflictedFiles, result.src)
			case Archived:
				copiedNum += 1
				fmt.Printf("%s -> %s\n", result.src, result.dst)
			case IgnoreErrorTime:
				errorTimeFiles = append(errorTimeFiles, result.src)
			case IgoreExisted:
				duplicated += 1

			}
			// fmt.Printf("%v %s archived\n", result.created, result.path)
			// path, err := result.path, result.err

		}

		fmt.Printf("============ Summary ============\n")
		fmt.Printf("Files copied: %d\n", copiedNum)
		fmt.Printf("Files duplicated: (%d): \n", duplicated)
		fmt.Printf("Files with no time(%d): \n", len(errorTimeFiles))
		for _, path := range errorTimeFiles {
			fmt.Printf("%s\n", path)
		}
		fmt.Printf("Files copy failed(%d): \n", len(copyFailed))
		for _, path := range copyFailed {
			fmt.Fprintf(os.Stdout, "%s\n", path)
		}
		fmt.Fprintf(os.Stdout, "Files conflicted(%d): \n", len(conflictedFiles))
		for _, path := range conflictedFiles {
			fmt.Fprintf(os.Stdout, "%s\n", path)
		}

	}()

	elapsed := time.Since(start)
	log.Printf("took %s", elapsed)
}

var concurrentNum = 1
var moveFlag = false

func main() {
	flag.IntVar(&concurrentNum, "c", 1, "concurrent number")
	flag.BoolVar(&moveFlag, "move", false, "moving instead of copying")
	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Concurrent number: %d\n", concurrentNum)

	archive(flag.Args()[0], flag.Args()[1])
}
