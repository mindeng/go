package minlib

import (
	"fmt"
	"io"
	"io/ioutil"
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

	// if _, err := os.Stat(dst); err == nil {
	// 	// dst is existed
	// 	return ErrFileExists{path: dst}
	// }

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

	return os.Rename(tmp.Name(), dst)
}
