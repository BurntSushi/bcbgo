package util

import (
	"bufio"
	"io"
	"strings"
)

func ReadLines(r io.Reader) []string {
	buf := bufio.NewReader(r)
	lines := make([]string, 0)
	for {
		line, err := buf.ReadString('\n')
		if err != nil && err != io.EOF {
			Fatalf("Could not read line: %s.", err)
		}
		lines = append(lines, strings.TrimSpace(line))
		if err == io.EOF {
			break
		}
	}
	return lines
}

func CopyFile(src, dest string) {
	_, err := io.Copy(CreateFile(dest), OpenFile(src))
	Assert(err, "Could not copy '%s' to '%s'", src, dest)
}
