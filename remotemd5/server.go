package remotemd5

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/mindeng/go/minlib"
)

type Server struct {
	listen  net.Listener
	rootDir string
}

func (s *Server) Run(host string, port string, rootDir string) {
	s.rootDir = rootDir

	var err error
	s.listen, err = net.Listen("tcp", host+":"+port)
	if err != nil {
		log.Fatalln("Error listening:", err)
	}
	defer s.shutdown()

	s.acceptConnections()
}

func (s *Server) shutdown() {
	if s.listen != nil {
		s.listen.Close()
		s.listen = nil
	}
}

func (s *Server) acceptConnections() {
	for {
		conn, err := s.listen.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			os.Exit(1)
		}
		//logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())

		// Handle connections in a new goroutine.
		go s.handleRequest(conn)
	}
}

func (s *Server) doJob(jobs <-chan string, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range jobs {
		// dst := filepath.Join(s.rootDir, path)
		dst := path
		checksum, err := minlib.FileChecksum(dst)
		if err != nil {
			log.Fatal(err)
		}

		log.Println(path, checksum)

		results <- Result{path: path, md5: checksum}
	}
}

func handleSendResponse(conn net.Conn, results <-chan Result, wg *sync.WaitGroup) {
	for result := range results {
		msg := fmt.Sprintf("%s\t%v\n", result.path, result.md5)

		if _, err := conn.Write([]byte(msg)); err != nil {
			log.Fatal(err)
		}
	}
	wg.Done()
}

func (s *Server) handleRequest(conn net.Conn) {
	defer conn.Close()

	jobs := make(chan string, 100)
	results := make(chan Result, 100)

	const concurrentNum = 4
	var wg sync.WaitGroup
	wg.Add(concurrentNum)
	for i := 0; i < concurrentNum; i++ {
		go s.doJob(jobs, results, &wg)
	}

	var wgSend sync.WaitGroup
	wgSend.Add(1)
	go handleSendResponse(conn, results, &wgSend)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		} else if strings.TrimSpace(line) == "." {
			// all jobs have been received
			close(jobs)

			// wait for all jobs done
			wg.Wait()

			// all results have been sent
			close(results)

			// wait for all results sent
			wgSend.Wait()

			// send done signal
			result := fmt.Sprintf(".\n")
			log.Printf("%s done\n", conn.RemoteAddr())
			if _, err := conn.Write([]byte(result)); err != nil {
				log.Fatal(err)
			}
			break
		}

		jobs <- line
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
