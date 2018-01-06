// +build exif

package minlib

import (
	"fmt"
	"testing"
	"time"
)

func TestFileOriginalTime(t *testing.T) {
	cases := []struct {
		path         string
		originalTime time.Time
	}{
		{"/Users/min/Pictures/t1.mov", time.Now()},
	}

	fmt.Println(FileOriginalTime(cases[0].path))
}

func TestAVIFileOriginalTime(t *testing.T) {
	cases := []struct {
		path         string
		originalTime time.Time
	}{
		{"/Users/min/Pictures/照片图库.photoslibrary/Masters/2016/01/20/20160120-030700/DSCN3717.AVI", time.Now()},
	}

	fmt.Println(FileOriginalTime(cases[0].path))
}
