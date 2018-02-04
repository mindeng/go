package main

import (
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

type ArchiveResultType int

const (
	Archived ArchiveResultType = iota
	CopyFailed
	IgnoreExisted
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
type CompareFileTask struct {
	src        string
	srcCreated time.Time
	dst        string
	result     bool
}

func startCopyFileService(tasks <-chan CopyFileTask, results chan<- ArchiveResult) *sync.WaitGroup {
	var wg sync.WaitGroup
	const concurrentNum = 1
	wg.Add(concurrentNum)

	copy := func() {
		defer wg.Done()
		for t := range tasks {
			result := CopyFailed
			var err error
			if dstInfo, err := os.Stat(t.dst); err == nil {
				if srcInfo, err := os.Stat(t.src); err == nil && srcInfo.Size() == dstInfo.Size() {
					if minlib.EqualFile(t.src, t.dst) {
						result = IgnoreExisted
						goto RESULT
					}
				}
				checksum, err := minlib.FileChecksum(t.src)
				if err != nil {
					goto RESULT
				}
				t.dst = uniqueFilename(t.dst, checksum)
			}
			if moveFlag {
				err = os.Rename(t.src, t.dst)
			} else {
				err = minlib.CopyFile(t.dst, t.src)
			}

			if err == nil {
				result = Archived
			}

		RESULT:
			results <- ArchiveResult{t.src, t.dst, result, err}
		}
	}

	for i := 0; i < concurrentNum; i++ {
		go copy()
	}

	return &wg
}

func startExtractFileTimeService(tasks <-chan string, results chan<- MediaInfo) *sync.WaitGroup {
	var wg sync.WaitGroup
	const concurrentNum = 2
	wg.Add(concurrentNum)

	extractTime := func() {
		defer wg.Done()
		for path := range tasks {
			created, err := minlib.FileTime(path)
			results <- MediaInfo{path, created, err}
		}
	}

	for i := 0; i < concurrentNum; i++ {
		go extractTime()
	}

	return &wg
}

type CompareCallback func(task CompareFileTask)

func startCompareFileServiceLocally(tasks <-chan CompareFileTask, cb CompareCallback) *sync.WaitGroup {
	var wg sync.WaitGroup
	const concurrentNum = 2
	wg.Add(concurrentNum)

	compareFile := func() {
		defer wg.Done()
		for task := range tasks {

			src := task.src
			dst := task.dst
			task.result = false

			if info, err := os.Stat(dst); err == nil {
				t, _ := minlib.FileTime(dst)
				if t == task.srcCreated {
					if srcInfo, err := os.Stat(src); err == nil && srcInfo.Size() == info.Size() {
						task.result = minlib.EqualFile(src, dst)
					}
				}
			} else {
				log.Fatalf("File not exists: %s\n", dst)
			}
			cb(task)
		}
	}

	for i := 0; i < concurrentNum; i++ {
		go compareFile()
	}

	return &wg
}

type CompareFileInfo struct {
	src         string
	dst         string
	srcChecksum string
}

func uniqueFilename(filename string, checksum string) string {
	ext := filepath.Ext(filename)
	return fmt.Sprintf("%s-%s%s", filename[:len(filename)-len(ext)], checksum[:6], ext)
}

func startCompareFileServiceRemote(tasks <-chan CompareFileTask, copyTasks chan<- CopyFileTask, cb CompareCallback) *sync.WaitGroup {
	var wg sync.WaitGroup
	const concurrentNum = 2
	wg.Add(concurrentNum)

	var wgRet sync.WaitGroup
	wgRet.Add(1)

	var lock = sync.RWMutex{}
	pathMap := make(map[string]CompareFileInfo)

	copyConflictedFile := func(src string, dst string, srcChecksum string) {
		// conflicted, copy it with another name
		dst = uniqueFilename(dst, srcChecksum)
		copyTasks <- CopyFileTask{src, dst}
	}

	var remoteTasks chan VerifyFileChecksumTask
	remoteTasks = make(chan VerifyFileChecksumTask, 100)
	startRemoteVerifyService(*host, *port, remoteTasks, func(task VerifyFileChecksumTask) {
		if task.checksum == "done" {
			wgRet.Done()
		} else {
			lock.RLock()
			srcDst := pathMap[task.path]
			lock.RUnlock()

			if task.result {
				cb(CompareFileTask{srcDst.src, time.Time{}, srcDst.dst, task.result})
			} else {
				// conflicted, copy it with another name
				copyConflictedFile(srcDst.src, srcDst.dst, srcDst.srcChecksum)
			}
		}
	})

	compareFile := func() {
		defer wg.Done()

		for task := range tasks {

			src := task.src
			dst := task.dst
			task.result = false

			if info, err := os.Stat(dst); err == nil {
				// dst exists

				checksum, err := minlib.FileChecksum(src)
				if err != nil {
					log.Fatalln("calc checksum error: ", err)
				}

				t, _ := minlib.FileTime(dst)
				if t == task.srcCreated {
					if srcInfo, err := os.Stat(src); err == nil && srcInfo.Size() == info.Size() {

						remoteDst := dst[len(flag.Args()[1]):]
						remoteDst = strings.TrimLeft(remoteDst, "/")
						lock.Lock()
						pathMap[remoteDst] = CompareFileInfo{src, dst, checksum}
						lock.Unlock()
						remoteTasks <- VerifyFileChecksumTask{remoteDst, checksum, false}
						continue
					}
				}

				// conflicted, copy it with another name
				copyConflictedFile(src, dst, checksum)
			} else {
				log.Fatalf("File not exists: %s\n", dst)
			}
		}
	}

	for i := 0; i < concurrentNum; i++ {
		go compareFile()
	}

	go func() {
		wg.Wait()
		close(remoteTasks)
	}()

	return &wgRet
}

func archiveFiles(mediaFiles <-chan MediaInfo, dstDir string, results chan<- ArchiveResult) {
	copyTasks := make(chan CopyFileTask, 100)
	compareTasks := make(chan CompareFileTask, 100)

	copyWait := startCopyFileService(copyTasks, results)

	var compareWait *sync.WaitGroup
	compareCb := func(task CompareFileTask) {
		if task.result {
			results <- ArchiveResult{task.src, task.dst, IgnoreExisted, nil}
		} else {
			// results <- ArchiveResult{task.src, task.dst, CopyConflict, errors.New("file conflicted")}
			log.Fatalf("File conflicted: %s %s\n", task.src, task.dst)
		}
	}
	if *remote {
		compareWait = startCompareFileServiceRemote(compareTasks, copyTasks, compareCb)
	} else {
		compareWait = startCompareFileServiceLocally(compareTasks, compareCb)
	}

	for mediaFile := range mediaFiles {
		src := mediaFile.path
		// log.Println("got ", src)
		created := mediaFile.created

		dst := path.Join(dstDir, fmt.Sprintf("%02d/%02d/%02d", created.Year(), created.Month(), created.Day()), path.Base(src))
		dstDir := path.Dir(dst)
		if _, err := os.Stat(dstDir); os.IsNotExist(err) {
			if err = os.MkdirAll(dstDir, 0755); err != nil {
				results <- ArchiveResult{src, dst, CopyFailed, err}
				break
			}
		}

		if _, err := os.Stat(dst); err != nil {
			// dst not exists, copy the file
			copyTasks <- CopyFileTask{src, dst}
		} else {
			// dst exists, compare src & dst
			compareTasks <- CompareFileTask{src, created, dst, false}
		}
	}

	// should close & wait compare tasks first, because compare task may add new copy task
	close(compareTasks)
	compareWait.Wait()
	fmt.Println("compare tasks done")

	close(copyTasks)
	copyWait.Wait()
	fmt.Println("copy tasks done")

	close(results)
}

var mediaFilesNum = 0

func walkDirectory(dir string, results chan MediaInfo) {
	tasks := make(chan string, 100)
	wg := startExtractFileTimeService(tasks, results)

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
			mediaFilesNum += 1
			tasks <- path
			return nil
		default:
			return nil
		}
	})

	close(tasks)

	wg.Wait()

	// log.Println("q closed")
	close(results)
}

func archive(src string, dst string) {
	mediaFiles := make(chan MediaInfo, 100)
	results := make(chan ArchiveResult, 1000)
	go walkDirectory(src, mediaFiles)
	go archiveFiles(mediaFiles, dst, results)

	start := time.Now()

	func() {
		var copyFailed []string

		copiedNum, duplicated := 0, 0
		for result := range results {
			if result.err != nil {
				fmt.Fprintf(os.Stderr, "[error] %v: %s %s\n", result.err, result.src, result.dst)
			}

			switch result.result {
			case CopyFailed:
				copyFailed = append(copyFailed, result.src)
			case Archived:
				copiedNum += 1
				fmt.Printf("cp %s %s\n", result.src, result.dst)
			case IgnoreExisted:
				duplicated += 1

			}
			// fmt.Printf("%v %s archived\n", result.created, result.path)
			// path, err := result.path, result.err

		}

		fmt.Printf("============ Summary ============\n")
		fmt.Printf("Files number: %d\n", mediaFilesNum)
		fmt.Printf("Files copied: %d\n", copiedNum)
		fmt.Printf("Files duplicated: %d: \n", duplicated)
		fmt.Printf("Files copy failed(%d): \n", len(copyFailed))
		for _, path := range copyFailed {
			fmt.Fprintf(os.Stdout, "%s\n", path)
		}
		fmt.Printf("Files unknown: %d\n", mediaFilesNum-copiedNum-duplicated-len(copyFailed))

	}()

	elapsed := time.Since(start)
	log.Printf("took %s", elapsed)
}

var concurrentNum = 1
var moveFlag = false

var host = flag.String("host", "localhost", "host")
var port = flag.String("port", "3333", "port")
var remote = flag.Bool("remote", false, "remote compare")

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
