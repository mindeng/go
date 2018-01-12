package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/mindeng/go/remotemd5"
)

var host = flag.String("host", "", "host")
var port = flag.String("port", "3333", "port")

func main() {
	flag.Parse()

	files := make(chan string, 100)
	md5OfFiles := make(chan remotemd5.Result, 100)

	client := &remotemd5.Client{}
	client.Connect(*host, *port)
	client.Process(files, md5OfFiles)

	root := flag.Arg(0)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		files <- path
		return nil
	})

	close(files)
	client.Wait()

}
