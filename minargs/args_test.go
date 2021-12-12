package minargs

import (
	"flag"
	"log"
	"testing"
)

var checkFlag StringArg

func init() {
	flag.Var(&checkFlag, "check", "check flag")
}

func TestStringArg(t *testing.T) {
	log.Println(checkFlag.Provided())
	if !checkFlag.Provided() {
		t.Fail()
	}
}
