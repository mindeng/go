package main

import "github.com/mindeng/go/minlib"
import "os"
import "fmt"

func main() {
	f1 := os.Args[1]
	f2 := os.Args[2]
	if !minlib.EqualFile(f1, f2) {
		fmt.Printf("Files differ: %s %s\n", f1, f2)
	}
}
