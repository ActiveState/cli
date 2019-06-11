package projectfile

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
)

type yamlReader struct {
	io.Reader
}

func (r *yamlReader) replaceInValue(key FileKey, old, new string) (io.Reader, *failures.Failure) {
	if key == "" {
		return r, failures.FailDeveloper.New("key must not be empty")
	}
	if old == "" {
		return r, failures.FailDeveloper.New("old value must not be empty")
	}
	if new == "" {
		return r, failures.FailDeveloper.New("new value must not be empty")
	}

	buf := &bytes.Buffer{}
	sc := bufio.NewScanner(r)

	for sc.Scan() {
		l := sc.Text()
		if !yamlLineHasKeyPrefix(l, string(key)) {
			if _, err := buf.WriteString(l + "\n"); err != nil {
				return nil, failures.FailIO.Wrap(err)
			}
			continue
		}

		l = replaceInYAMLValue(l, old, new)
		if _, err := buf.WriteString(l + "\n"); err != nil {
			return nil, failures.FailIO.Wrap(err)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	return buf, nil
}

func overwriteFile(f *os.File, r io.Reader) error {
	if err := f.Truncate(0); err != nil {
		return err
	}

	_, err := io.Copy(f, r)
	return err
}

func yamlLineHasKeyPrefix(line, key string) bool {
	spl := strings.SplitN(line, ":", 2)
	if len(spl) < 2 {
		return false
	}

	front := strings.TrimSpace(spl[0])

	return strings.HasPrefix(front, key)
}

func replaceInYAMLValue(line, old, new string) string {
	spl := strings.SplitN(line, ":", 2)
	if len(spl) < 2 {
		return line
	}

	front := spl[0]
	back := strings.Replace(spl[1], old, new, 1)

	return front + ":" + back
}
