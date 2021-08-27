package util

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/subosito/gotenv"
)

// LoadEnvFromFile - Loads the environment variables from a file
func LoadEnvFromFile(filename string) error {
	if filename != "" {
		// have to do filtering before using the gotenv library. While the library trims trailing
		// whitespace, a TAB char is no considered whitespace and isn't trimmed. Otherwise, we
		// just could have called gotenv.Load and skipped all of this.
		f, err := os.Open(filename)
		if err != nil {
			return err
		}

		defer f.Close()

		buf := filterLines(f)
		r := bytes.NewReader(buf.Bytes())

		return gotenv.Apply(r)
	}

	return nil
}

func filterLines(r io.Reader) bytes.Buffer {
	var lf = []byte("\n")

	var out bytes.Buffer
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		// trim out trailing spaces AND tab chars
		trimmedLine := strings.TrimRight(line, " \t")
		out.Write([]byte(trimmedLine))
		out.Write(lf)
	}

	return out
}
