package minlib

import (
    "testing"
    "math/rand"
    "time"
    "io/ioutil"
    "os"
    "bytes"
)

func init() {
    rand.Seed(time.Now().UnixNano())
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ你")

func RandString(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}

func TestCopyFile(t *testing.T)  {
    for i := 0; i < 10; i++ {
        size := (i+1) * 1024
        text := RandString(size)
        src, err := ioutil.TempFile("", "")
        if err != nil {
            t.Error(err)
            return
        }
        
        releaseFile := func (f *os.File) {
            if f == nil {
                return
            }
            f.Close()
            os.Remove(f.Name())
        }

        defer releaseFile(src)

        if err != nil {
            t.Error(err)
            return
        }
        if _, err = src.WriteString(text); err != nil {
            t.Error(err)
            return
        }

        if err = src.Close(); err != nil {
            t.Error(err)
            return
        }

        dst, err := ioutil.TempFile("", "")
        if err != nil {
            t.Error(err)
            return
        }
        defer releaseFile(dst)

        // Change modification time for file src
        twoDaysFromNow := time.Date(1988, 1, 1, 0, 0, 0, 0, time.Local)
        lastAccessTime := twoDaysFromNow
        lastModifyTime := twoDaysFromNow    
        err = os.Chtimes(src.Name(), lastAccessTime, lastModifyTime)
        if err != nil {
            t.Error(err)
            return
        }

        if err = CopyFile(dst.Name(), src.Name()); err != nil {
            t.Error(err)
            return
        }

        dst.Close()

        dst, err = os.Open(dst.Name())
        if err != nil {
            t.Error(err)
            return
        }
    
        dstStat, err := os.Stat(dst.Name())
        if err != nil {
            t.Error(err)
            return
        }
        buf := make([]byte, dstStat.Size())
        if _, err := dst.Read(buf); err != nil {
            t.Error(err)
            return
        }

        // Test modification time
        if dstStat.ModTime() != lastModifyTime {
            t.Errorf("%v != %v", dstStat.ModTime(), lastModifyTime)
            return
        }

        // Test file content
        if bytes.Compare([]byte(text), buf) != 0 {
            t.Errorf("[%v] != [%v]", []byte(text), buf)
            return
        }
    }
}