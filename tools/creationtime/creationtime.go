package main

import (
	"fmt"
	"github.com/mindeng/go/minlib"
	"os"
)

func main() {
	path := os.Args[1]
	ctime, err := minlib.FileOriginalTime(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", err, path)
		os.Exit(1)
	}
	fmt.Printf("%v %v\n", ctime, path)
}
