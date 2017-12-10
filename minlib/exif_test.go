// +build exif

package minlib

import (
    "testing"
    "time"
    "fmt"
)

func TestFileOriginalTime(t *testing.T)  {
    cases := []struct {
        path string
        originalTime time.Time
    } {
        {"/Users/min/Pictures/t1.mov", time.Now()},
    }

    fmt.Println(FileOriginalTime(cases[0].path))
}