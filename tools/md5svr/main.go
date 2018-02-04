package main

import (
	"flag"

	"github.com/mindeng/go/remotemd5"
)

var host = flag.String("host", "", "host")
var port = flag.String("port", "3333", "port")

func main() {
	flag.Parse()

	svr := &remotemd5.Server{}
	svr.Run(*host, *port, flag.Arg(0))
}
