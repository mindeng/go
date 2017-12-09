package main

import (
	"os"
	"fmt"
	"github.com/mindeng/go/minlib"
)

func main() {
	src, dst := os.Args[1], os.Args[2]
	if err := minlib.CopyFile(dst, src); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Copy file %v to %v completed.\n", src, dst)
}