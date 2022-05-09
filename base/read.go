package base

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// ReadText reads content from the given file with leading and tailing spaces trimmed.
func ReadText(filename string) (string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

func ReadFile(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("os/stat: read file(%s) failed!, e:%s", path, err.Error())
	}
	return strings.TrimSpace(string(contents)), nil
}

// ReadLines reads contents from a file and splits them by new lines.
// A convenience wrapper to ReadLinesOffsetN(filename, 0, -1).
func ReadLines(filename string) ([]string, error) {
	return readLinesOffsetN(filename, 0, -1)
}

// ReadLinesOffsetN reads contents from file and splits them by new line.
// The offset tells at which line number to start.
// The count determines the number of lines to read (starting from offset):
//   n >= 0: at most n lines
//   n < 0: whole file
func readLinesOffsetN(filename string, offset uint, n int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for i := 0; i < n+int(offset) || n < 0; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		if i < int(offset) {
			continue
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}

	return ret, nil
}
