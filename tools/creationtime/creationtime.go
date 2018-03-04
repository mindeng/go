package main

import (
	"fmt"
	"os"

	"github.com/mindeng/goutils"
)

func main() {
	path := os.Args[1]
	ctime, err := goutils.FileOriginalTime(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", err, path)
		os.Exit(1)
	}
	fmt.Printf("%v %v\n", ctime, path)
}
