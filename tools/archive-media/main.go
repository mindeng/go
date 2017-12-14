package main

import (
	"fmt"
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

func ArchiveDirectory(src string, archiveFunc ArchiveFunc, cb ArchiveCallback) error {
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			cb(path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		// if info.Size() <= 1024 {
		// 	cb(path, ErrArchiveIgnore{"ignore small file"})
		// 	return nil
		// }
		if info.Name()[0] == '.' && info.Size() == 4096 {
			return nil
		}
		switch strings.ToLower(filepath.Ext(info.Name())) {
		case ".jpg", ".png", ".arw", ".nef", ".avi", ".mp4", ".mov", ".m4v":
			// need to archive
		default:
			return nil
		}
		created, err := minlib.FileOriginalTime(path)
		if err != nil {
			cb(path, err)
			return nil
		}
		err = archiveFunc(path, created, info)
		cb(path, err)
		return nil
	})

	return nil
}

func main() {
	ArchiveDirectory(os.Args[1], func(path string, created time.Time, info os.FileInfo) error {
		if created.Year() < 1980 {
			fmt.Fprintf(os.Stderr, "time error: %v %s\n", created, path)
		} else {
			fmt.Printf("archive %v %s\n", created, path)
		}
		return nil
	}, func(path string, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed: %v %s\n", err, path)
		}
		return nil
	})
}
