package main

import (
	"os"
	"fmt"
	"github.com/mindeng/go/minlib"
	"log"
)

func main() {
	path := os.Args[1]
	ctime, err := minlib.FileOriginalTime(path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Creation time: %v .\n", ctime)
}