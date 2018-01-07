package main

import (
	"bufio"
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
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
func handleRequest(conn net.Conn) {
	defer conn.Close()

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
			// done
			result := fmt.Sprintf("done\tdone\n")
			fmt.Println("done")
			conn.Write([]byte(result))
			continue
		}

		path, checksum := params[0], params[1]

		dst := filepath.Join(flag.Args()[0], path)

		myChecksum, err := fileChecksum(dst)
		if err != nil {
			log.Fatal(err)
		}

		ok := "miss"
		if checksum == myChecksum {
			ok = "ok"
		}
		result := fmt.Sprintf("%s\t%v\n", path, ok)
		fmt.Println(line, " ", ok)
		conn.Write([]byte(result))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// for {
	// 	io.Copy(conn, conn)
	// }
}

func fileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	defer f.Close()

	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", md5.Sum(data)), nil
}
