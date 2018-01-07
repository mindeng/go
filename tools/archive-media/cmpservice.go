package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type VerifyFileChecksumTask struct {
	path     string
	checksum string
	result   bool
}

type VerifyCallback func(verifyTask VerifyFileChecksumTask)

var verifyTaskChan = make(chan VerifyFileChecksumTask, 100)
var verifyCallback VerifyCallback

var conn net.Conn

func startRemoteVerifyService(host string, port string, tasks <-chan VerifyFileChecksumTask, cb VerifyCallback) {
	verifyCallback = cb

	c, err := net.Dial("tcp", host+":"+port)
	conn = c
	if err != nil {
		log.Fatalln("Connect remote verify service error: ", err)
	}

	go handleResponse()

	go func() {
		for task := range tasks {
			sendTask(task)
		}
		sendTask(VerifyFileChecksumTask{"done", "done", false})
	}()

}

func sendTask(task VerifyFileChecksumTask) {
	msg := fmt.Sprintf("%s\t%s\n", task.path, task.checksum)
	data := []byte(msg)
	n, err := conn.Write(data)
	if err != nil || n != len(data) {
		log.Fatal(err)
	}
}

func handleResponse() {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString(byte('\n'))
		if err != nil {
			log.Fatal(err)
		}
		// log.Print(line)
		if strings.TrimSpace(line) == "" {
			continue
		}
		result := strings.Split(line, "\t")
		if len(result) != 2 {
			log.Fatalln("recv invlalid response: ", line)
		}

		ok := false
		if strings.TrimSpace(result[1]) == "ok" {
			ok = true
			verifyCallback(VerifyFileChecksumTask{result[0], "", ok})
		} else if strings.TrimSpace(result[1]) == "done" {
			// fmt.Println("recv done")
			verifyCallback(VerifyFileChecksumTask{"done", "done", false})
		}
	}
}
