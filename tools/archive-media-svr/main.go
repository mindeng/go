package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mindeng/go/minlib"
)

var host = flag.String("host", "", "host")
var port = flag.String("port", "3333", "port")

func main() {
	flag.Parse()
	var l net.Listener
	var err error
	l, err = net.Listen("tcp", *host+":"+*port)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer l.Close()
	fmt.Println("Listening on " + *host + ":" + *port)
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			os.Exit(1)
		}
		//logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}

type VerifyJob struct {
	path     string
	checksum string
	result   bool
}

func handleVerifyJob(jobs <-chan VerifyJob, results chan<- VerifyJob, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		dst := filepath.Join(flag.Args()[0], job.path)
		myChecksum, err := minlib.FileChecksum(dst)
		if err != nil {
			log.Fatal(err)
		}
		job.result = myChecksum == job.checksum
		results <- job
	}
}

func handleSendResponse(conn net.Conn, jobs <-chan VerifyJob, wg *sync.WaitGroup) {
	for job := range jobs {
		ok := "miss"
		if job.result {
			ok = "ok"
		}
		result := fmt.Sprintf("%s\t%v\n", job.path, ok)
		fmt.Print(result)
		conn.Write([]byte(result))
	}
	wg.Done()
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	jobs := make(chan VerifyJob, 100)
	results := make(chan VerifyJob, 100)

	const concurrentNum = 4
	var wg sync.WaitGroup
	wg.Add(concurrentNum)
	for i := 0; i < concurrentNum; i++ {
		go handleVerifyJob(jobs, results, &wg)
	}

	var wgSend sync.WaitGroup
	wgSend.Add(1)
	go handleSendResponse(conn, results, &wgSend)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		params := strings.Split(line, "\t")
		if len(params) < 2 {
			continue
		}

		if params[0] == "done" && params[1] == "done" {
			// all jobs have been received
			close(jobs)

			// wait for all jobs done
			wg.Wait()

			// all results have been sent
			close(results)

			// wait for all results sent
			wgSend.Wait()

			// send done signal
			result := fmt.Sprintf("done\tdone\n")
			fmt.Print(result)
			conn.Write([]byte(result))
			break
		}

		path, checksum := params[0], params[1]
		jobs <- VerifyJob{path, checksum, false}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
