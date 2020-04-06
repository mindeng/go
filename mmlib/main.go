package main

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/boltdb/bolt"
)

// Task is a task of calculating file's md5
type Task struct {
	path     string
	filesize int
	data     []byte
	md5      []byte
}

func doJob(jobs <-chan Task, results chan<- Task) {
	for job := range jobs {
		h := md5.New()

		h.Write(job.data)
		var md5 = h.Sum(nil)
		job.md5 = md5

		results <- job
	}

	fmt.Fprintf(os.Stderr, "calc done\n")
	close(results)
}

func readFile(filename string, maxsize int) ([]byte, int, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, -1, err
	}
	defer file.Close()

	stats, statsErr := file.Stat()
	if statsErr != nil {
		return nil, -1, statsErr
	}

	var size int64 = stats.Size()
	var bufsize = maxsize
	if size < int64(bufsize) {
		bufsize = int(size)
	}
	// fmt.Printf("bufsize: %v\n", bufsize)

	var data = make([]byte, bufsize)
	bufr := bufio.NewReader(file)
	_, err = bufr.Read(data)

	return data, int(size), err
}

func walkDirectory(dir string, tasks chan string) (int, int) {
	var total = 0
	var ignored = 0
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
			total++
			return nil
		default:
			fmt.Fprintf(os.Stderr, "ignore: %s\n", path)
			ignored++
			return nil
		}
	})

	close(tasks)

	// fmt.Fprintf(os.Stderr, "process: %d ignored: %d\n", total, ignored)
	return total, ignored
}

func save(results chan Task) error {
	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var result Task
	var ok bool = true
	var num = 0

	for ok {
		if err := db.Update(func(tx *bolt.Tx) error {
			b, _ := tx.CreateBucketIfNotExists([]byte("mm"))
			num = 0
			for {
				result, ok = <-results
				if !ok {
					break
				}

				fmt.Fprintf(os.Stdout, "md5 of %s: %x %d\n", result.path, result.md5, result.filesize)
				num++
				b.Put([]byte(fmt.Sprintf("%s:md5", result.path)), result.md5)
				bs := make([]byte, 4)
				binary.LittleEndian.PutUint32(bs, uint32(result.filesize))
				b.Put([]byte(fmt.Sprintf("%d:size", result.filesize)), bs)

				if v := b.Get(result.md5); v != nil {
					fmt.Fprintf(os.Stderr, "duplicated md5: %s %s %x\n", result.path, v, result.md5)
				} else {
					b.Put(result.md5, []byte(result.path))
				}

				if num >= 100 {
					break
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	var pathList = make(chan string)
	var processed, ignored int

	go func() {
		processed, ignored = walkDirectory(os.Args[1], pathList)
	}()

	jobs := make(chan Task, 10)
	results := make(chan Task, 100)

	go doJob(jobs, results)

	go func() {
		for path := range pathList {
			data, size, err := readFile(path, 1<<20)
			if err != nil {
				fmt.Fprintf(os.Stderr, "read file error: %s %v\n", path, err)
				continue
			}

			jobs <- Task{path: path, filesize: size, data: data}
		}

		fmt.Fprintf(os.Stderr, "read done\n")
		close(jobs)
	}()

	// go func() {
	// for result := range results {
	// 	fmt.Fprintf(os.Stdout, "md5 of %s: %x %d\n", result.path, result.md5, result.filesize)
	// }
	// }()

	save(results)

	fmt.Fprintf(os.Stderr, "processed: %d ignored: %d\n", processed, ignored)
}
