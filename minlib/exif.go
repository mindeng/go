package minlib

import (
	"time"
	"path/filepath"
	"strings"
	"os"
	"bytes"
	"encoding/binary"
	"errors"
)

// FileOriginalTime returns the original time for file p.
func FileOriginalTime(p string) (originalTime time.Time, err error) {
	ext := strings.ToLower(filepath.Ext(p))
	switch (ext) {
	case ".mov", ".mp4":
		originalTime, err = movOriginalTime(p)
		return
	}
	return 
}

func movOriginalTime(p string) (originalTime time.Time, err error)  {
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
			in.Seek(atomSize - 8, 1)
		}
	}

	// found 'moov', look for 'mvhd' and timestamps
	_, err = in.Read(atomHeader)
	if err != nil {
		return
	}
	if bytes.Compare(atomHeader[4:8], []byte("cmov")) == 0 {
		err = errors.New("moov atom is compressed")
		return
	} else if bytes.Compare(atomHeader[4:8], []byte("mvhd")) != 0 {
		err = errors.New("expected to find 'mvhd' header")
		return
	} else {
		in.Seek(4, 1)
		if _, err = in.Read(dword); err != nil {
			return
		}
		timestamp := int64(binary.BigEndian.Uint32(dword))
		timestamp -= int64(EPOCH_ADJUSTER)
		originalTime = time.Unix(timestamp, 0)
		
		// if _, err = in.Read(dword); err != nil {
		// 	return nil, err
		// }
		// modificationDate := time.Unix(int64(binary.BigEndian.Uint32(dword[0:4])), 0)

		return
	}
}