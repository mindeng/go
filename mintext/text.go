package mintext

import (
	"bufio"
	"io"
	"strings"
)

// 通过 token 按行搜索，例如：
// 假设 line = "hash: [xxx] path: [yyy]"
// 则：
// 	  1. tokens = "[", "]", "[", "]"
//       返回 ["xxx", " path: ", "yyy"]
//    2. tokens = "[", "] path: [", "]"
//       返回 ["xxx", "yyy"]
func SearchInLine(r io.Reader, tokens ...string) <-chan []string {
	out := make(chan []string)

	go func() {
		defer close(out)
		s := bufio.NewScanner(r)
		s.Split(bufio.ScanLines)
		for s.Scan() {
			data := s.Bytes()
			if len(data) == 0 {
				continue
			}
			line := strings.TrimSpace(string(data))
			if len(line) == 0 {
				continue
			}

			out <- extract(line, tokens...)
		}
	}()

	return out
}

func extract(line string, tokens ...string) []string {
	ss := make([]string, 0)

	for i := range tokens {
		if i == len(tokens)-1 {
			break
		}

		token1 := tokens[i]
		token2 := tokens[i+1]
		s := strBetweenToken(line, token1, token2)

		ss = append(ss, s)
	}

	return ss
}

func strBetweenToken(line, token1, token2 string) string {
	start := strings.Index(line, token1)
	if start == -1 {
		return ""
	}
	start += len(token1)

	end := strings.Index(line[start:], token2)
	if end == -1 {
		return ""
	}
	return line[start : start+end]
}
