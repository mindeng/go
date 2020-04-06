package main

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
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

func makeMd5Key(path string) []byte {
	return []byte(fmt.Sprintf("%s:md5", path))
}

var makeSizeKey = func(path string) []byte {
	return []byte(fmt.Sprintf("%s:size", path))
}

func calcMd5(jobs <-chan Task, results chan<- Task, db *bolt.DB) {
	for job := range jobs {
		var sum []byte

		if db != nil {
			db.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("mm"))
				sum = b.Get(makeMd5Key(job.path))
				return nil
			})
		}

		if sum == nil {
			h := md5.New()

			h.Write(job.data)
			sum = h.Sum(nil)
			fmt.Fprintf(os.Stdout, "md5: %s %x\n", job.path, sum)
		}
		job.md5 = sum

		results <- job
	}

	fmt.Fprintf(os.Stdout, "calc done\n")
	close(results)
}

func getFileSize(path string) (int64, error) {
	file, err := os.Open(path)

	if err != nil {
		return 0, err
	}
	defer file.Close()

	var stats os.FileInfo
	if stats, err = file.Stat(); err != nil {
		return 0, err
	}

	return stats.Size(), nil
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

func walkDirectory(dir string, tasks chan string, db *bolt.DB) (int, map[string]int) {
	var processed = 0
	var ignoredExts = make(map[string]int)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v %s\n", err, path)
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

		ext := strings.ToLower(filepath.Ext(info.Name()))
		// fmt.Printf("ext: %s\n", ext)
		switch ext {
		case ".jpg", ".jpeg", ".png", ".arw", ".nef", ".avi", ".mp4", ".mov", ".m4v", ".m4a", ".gif":
			// need to archive
			// log.Println("put ", path)

			var exists = false

			if err := db.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("mm"))
				if v := b.Get(makeMd5Key(path)); v != nil {
					if size, err := getFileSize(path); err == nil {
						if storedSize := b.Get(makeSizeKey(path)); storedSize != nil {
							var storedFileSize = binary.BigEndian.Uint32(storedSize)
							// fmt.Printf("size: %d storedSize: %d\n", size, storedFileSize)
							if size == int64(storedFileSize) {
								exists = true
							} else {
								fmt.Fprintf(os.Stdout, "updated: %s\n", path)
							}
						}
					}
					// fmt.Printf("md5: %x\n", v)
				}
				return nil
			}); err != nil {
				log.Fatal(err)
			}

			if exists {
				// ignore existed path
				fmt.Fprintf(os.Stdout, "exists: %s\n", path)
			} else {
				tasks <- path
				processed++
			}
			return nil
		default:
			// fmt.Fprintf(os.Stderr, "ignore: %s\n", path)
			ignoredExts[ext] = 1
			return nil
		}
	})

	close(tasks)

	// fmt.Fprintf(os.Stderr, "process: %d ignored: %d\n", total, ignored)
	return processed, ignoredExts
}

func save(db *bolt.DB, results chan Task) error {

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

				if v := b.Get(result.md5); v != nil {
					if size := b.Get(makeSizeKey(string(v))); size != nil {
						var filesize = binary.BigEndian.Uint32(size)
						if filesize == uint32(result.filesize) {
							fmt.Fprintf(os.Stderr, "duplicated: %s %s\n", result.path, v)
						} else {
							fmt.Fprintf(os.Stderr, "conflict: %s %s %x\n", result.path, v, result.md5)
						}
					}
				} else {
					fmt.Fprintf(os.Stdout, "add: %s %x %d\n", result.path, result.md5, result.filesize)
					b.Put(result.md5, []byte(result.path))
				}

				num++
				b.Put([]byte(fmt.Sprintf("%s:md5", result.path)), result.md5)
				bs := make([]byte, 4)
				binary.BigEndian.PutUint32(bs, uint32(result.filesize))
				b.Put(makeSizeKey(result.path), bs)

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

var outdbPath = flag.String("outdb", "", "output db path")
var indbPath = flag.String("indb", "", "input db path")

func init() {
	// init short version for long flag
	flag.StringVar(outdbPath, "o", "", "output db path")
	flag.StringVar(indbPath, "i", "", "input db path")
}

func main() {
	// var wg sync.WaitGroup
	// wg.Add(1)

	flag.Parse()
	var dstDir = flag.Args()[0]

	// fmt.Printf("outdbPath: %s\n", *outdbPath)

	if *outdbPath == "" {
		*outdbPath = path.Join(dstDir, "mm.db")
	}

	fmt.Printf("outdb: %s\n", *outdbPath)

	outdb, err := bolt.Open(*outdbPath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer outdb.Close()

	var indb *bolt.DB

	if *indbPath != "" {
		indb, err = bolt.Open(*indbPath, 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		defer indb.Close()
	}

	if err := outdb.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("mm"))
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	var pathList = make(chan string)
	var processed int
	var ignoredExts map[string]int

	go func(db *bolt.DB) {
		processed, ignoredExts = walkDirectory(dstDir, pathList, db)
	}(outdb)

	jobs := make(chan Task, 10)
	results := make(chan Task, 100)

	go func(db *bolt.DB) {
		calcMd5(jobs, results, db)
	}(indb)

	go func() {
		for path := range pathList {
			data, size, err := readFile(path, 1<<20)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s %v\n", path, err)
				continue
			}

			jobs <- Task{path: path, filesize: size, data: data}
		}

		fmt.Fprintf(os.Stdout, "read done\n")
		close(jobs)
	}()

	// go func() {
	// for result := range results {
	// 	fmt.Fprintf(os.Stdout, "md5 of %s: %x %d\n", result.path, result.md5, result.filesize)
	// }
	// }()

	// wg.Wait()

	if err := save(outdb, results); err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(os.Stdout, "processed: %d ignored: %v\n", processed, ignoredExts)
}
