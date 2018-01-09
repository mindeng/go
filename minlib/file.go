package minlib

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

type ErrFileExists struct {
	path string
}

func (err ErrFileExists) Error() string {
	return fmt.Sprint("file already exists: ", err.path)
}

// CopyFile copies the contents from src to dst atomically.
// If dst does not exist, CopyFile creates it and preserve the modification time.
// If the copy fails, CopyFile aborts and dst is preserved.
func CopyFile(dst, src string) error {
	fi, err := os.Stat(src)
	if os.IsNotExist(err) {
		// src is not existed
		return err
	}

	if _, err := os.Stat(dst); err == nil {
		// dst is existed
		return ErrFileExists{path: dst}
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// tmp, err := ioutil.TempFile("", "")
	tmp, err := ioutil.TempFile(filepath.Dir(dst), "_tmp_")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, in)
	if err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err = tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err = os.Chtimes(tmp.Name(), time.Now(), fi.ModTime()); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err = os.Chmod(tmp.Name(), fi.Mode()); err != nil {
		os.Remove(tmp.Name())
		return err
	}

	if _, err := os.Stat(dst); err == nil {
		// dst is existed
		return ErrFileExists{path: dst}
	}
	return os.Rename(tmp.Name(), dst)
}

const chunkSize = 64 * 1024

func EqualFile(file1, file2 string) bool {
	fi1, err := os.Stat(file1)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}
	fi2, err := os.Stat(file2)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	if fi1.Size() != fi2.Size() {
		return false
	}

	f1, err := os.Open(file1)
	if err != nil {
		log.Fatal(err)
	}
	defer f1.Close()

	f2, err := os.Open(file2)
	if err != nil {
		log.Fatal(err)
	}
	defer f2.Close()

	b1 := make([]byte, chunkSize)
	b2 := make([]byte, chunkSize)
	for {
		_, err1 := io.ReadFull(f1, b1)
		_, err2 := io.ReadFull(f2, b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true
			} else if err1 == io.EOF || err2 == io.EOF {
				return false
			} else if err1 == io.ErrUnexpectedEOF && err2 == io.ErrUnexpectedEOF {
				return bytes.Equal(b1, b2)
			} else if err1 == io.ErrUnexpectedEOF || err2 == io.ErrUnexpectedEOF {
				return false
			} else {
				log.Fatal(err1, err2)
			}
		}

		if !bytes.Equal(b1, b2) {
			return false
		}
	}
}

func FileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	defer f.Close()

	if err != nil {
		return "", nil
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", nil
	}

	return fmt.Sprintf("%x", md5.Sum(data)), nil
}
