package remotemd5

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

// Result is the calculating result
type Result struct {
	path string
	md5  string
	err  error
}

// Client send path to a remote md5 Server, and receive md5 of the file from the server
type Client struct {
	tasks   chan string
	results chan Result
	conn    net.Conn
	wg      *sync.WaitGroup
}

func (c *Client) Connect(host string, port string) error {
	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		return err
	}

	c.conn = conn
	c.wg = &sync.WaitGroup{}

	return nil
}

func (c *Client) Process(tasks chan string, results chan Result) {
	c.tasks = tasks
	c.results = results
	c.wg.Add(1)

	go c.processTasks()
}

func (c *Client) Wait() {
	c.wg.Wait()
}

func (c *Client) processTasks() {
	go c.handleResponse()

	for task := range c.tasks {
		c.sendTask(task)
	}
	c.sendTask(".")
}

func (c *Client) sendTask(path string) {
	data := []byte(path + "\n")
	n, err := c.conn.Write(data)
	if err != nil || n != len(data) {
		log.Fatal(err)
	}
}

func (c *Client) handleResponse() {
	defer c.conn.Close()
	reader := bufio.NewReader(c.conn)
	for {
		line, err := reader.ReadString(byte('\n'))
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		// log.Print(line)
		if strings.TrimSpace(line) == "" {
			continue
		} else if strings.TrimSpace(line) == "." {
			c.wg.Done()
			break
		}

		result := strings.Split(line, "\t")
		if len(result) != 2 {
			log.Fatalln("recv invlalid response: ", line)
		}

		path := result[0]
		md5 := strings.TrimSpace(result[1])

		c.results <- Result{path, md5, nil}
	}
}
