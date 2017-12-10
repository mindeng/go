package minlib

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type ErrNoOriginalTime struct {
	s string
}

func (err *ErrNoOriginalTime) Error() string {
	if len(err.s) != 0 {
		return fmt.Sprintf("original time not found: %s", err.s)
	} else {
		return fmt.Sprint("original time not found")
	}
}

// FileOriginalTime returns the original time for file p.
func FileOriginalTime(p string) (time.Time, error) {
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".mov", ".mp4":
		return movOriginalTime(p)
	default:
		return guessTimeFromFilename(p)
	}
}

func movOriginalTime(p string) (originalTime time.Time, err error) {
	ATOM_HEADER_SIZE := 8
	// difference between Unix epoch and QuickTime epoch, in seconds
	EPOCH_ADJUSTER := 2082844800
	// EPOCH_ADJUSTER := 0

	// open file and search for moov item
	in, err := os.Open(p)
	if err != nil {
		return
	}
	defer in.Close()

	atomHeader := make([]byte, ATOM_HEADER_SIZE)
	dword := make([]byte, 4)
	for {
		_, err = in.Read(atomHeader)
		if err != nil {
			return
		}
		if bytes.Compare(atomHeader[4:8], []byte("moov")) == 0 {
			break
		} else {
			atomSize := int64(binary.BigEndian.Uint32(atomHeader[0:4]))
			in.Seek(atomSize-8, 1)
		}
	}

	// found 'moov', look for 'mvhd' and timestamps
	_, err = in.Read(atomHeader)
	if err != nil {
		return
	}
	if bytes.Compare(atomHeader[4:8], []byte("cmov")) == 0 {
		err = &ErrNoOriginalTime{"moov atom is compressed"}
		return
	} else if bytes.Compare(atomHeader[4:8], []byte("mvhd")) != 0 {
		err = &ErrNoOriginalTime{"expected to find 'mvhd' header"}
		return
	} else {
		in.Seek(4, 1)
		if _, err = in.Read(dword); err != nil {
			return
		}
		timestamp := int64(binary.BigEndian.Uint32(dword))
		timestamp -= int64(EPOCH_ADJUSTER)
		if timestamp <= 0 {
			// fmt.Printf("guessTimeFromFilename: %s\n", p)
			return guessTimeFromFilename(p)
		}
		originalTime = time.Unix(timestamp, 0)

		// if _, err = in.Read(dword); err != nil {
		// 	return nil, err
		// }
		// modificationDate := time.Unix(int64(binary.BigEndian.Uint32(dword[0:4])), 0)

		return
	}
}

func guessTimeFromFilename(p string) (time.Time, error) {
	name := path.Base(p)
	var digits bytes.Buffer
	for _, c := range name {
		if c >= '0' && c <= '9' && digits.Len() < 14 {
			digits.WriteRune(c)
		}
	}

	if digits.Len() < 8 {
		return time.Time{}, &ErrNoOriginalTime{}
	}

	s := digits.String()
	s += " +0800"
	layout := "20060102150405 -0700"
	return time.Parse(layout, s)
}
