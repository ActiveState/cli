package execmeta

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

/*
::sock::/tmp/state-ipc/state-ipts.DX-1060.sock
::env::EXAMPLE=value::env::SAMPLE=other::env::THIRD=whatever
::bins::/example/bin/abc::bins::/example/bin/def::bins::/example/bin/xyz
::commit-id::1234abcd-1234-abcd-1234-abcd1234
::namespace::owner/name
::headless::true
*/

type Target struct {
	CommitUUID string
	Namespace  string
	Dir        string
	Headless   bool
}

var (
	MetaFileName = "meta.as"

	sockDelim      = "::sock::"
	envDelim       = "::env::"
	binsDelim      = "::bins::"
	commitDelim    = "::commit-id::"
	namespaceDelim = "::namespace::"
	headlessDelim  = "::headless::"
)

type ExecMeta struct {
	SockPath   string
	Env        map[string]string
	Bins       []string
	CommitUUID string
	Namespace  string
	Headless   bool
}

func New(sockPath string, env map[string]string, t Target, bins []string) *ExecMeta {
	return &ExecMeta{
		SockPath:   sockPath,
		Env:        env,
		Bins:       bins,
		CommitUUID: t.CommitUUID,
		Namespace:  t.Namespace,
		Headless:   t.Headless,
	}
}

func NewFromReader(r io.Reader) (*ExecMeta, error) {
	m := ExecMeta{}

	scnr := bufio.NewScanner(r)
	iter := -1
	for scnr.Scan() {
		iter++
		txt := scnr.Text()

		switch iter {
		case 0:
			m.SockPath = strings.TrimPrefix(txt, sockDelim)
		case 1:
			envMap := make(map[string]string)
			envTxt := strings.TrimPrefix(txt, envDelim)
			envSplit := strings.Split(envTxt, envDelim)
			for _, kv := range envSplit {
				kvSplit := strings.SplitN(kv, "=", 2)
				if len(kvSplit) < 2 {
					return nil, errors.New("env data malformed")
				}
				envMap[kvSplit[0]] = kvSplit[1]
			}
			m.Env = envMap
		case 2:
			binsTxt := strings.TrimPrefix(txt, binsDelim)
			m.Bins = strings.Split(binsTxt, binsDelim)
		case 3:
			m.CommitUUID = strings.TrimPrefix(txt, commitDelim)
		case 4:
			m.Namespace = strings.TrimPrefix(txt, namespaceDelim)
		case 5:
			boolTxt := strings.TrimPrefix(txt, headlessDelim)
			headless, err := strconv.ParseBool(boolTxt)
			if err != nil {
				return nil, err
			}
			m.Headless = headless
		default:
			return nil, errors.New("unexpected line in meta file")
		}

	}
	if err := scnr.Err(); err != nil {
		return nil, err
	}

	return &m, nil
}

// NewFromFile is a convenience func, not intended to be tested.
func NewFromFile(path string) (*ExecMeta, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(data)
	return NewFromReader(buf)
}

func (m *ExecMeta) WriteTo(w io.Writer) (int64, error) {
	aw := newAccumulatingWrite(w)

	aw.fprintf("%s%s\n", sockDelim, m.SockPath)
	for k, v := range m.Env {
		aw.fprintf("%s%s=%s", envDelim, k, v)
	}
	aw.fprintf("\n")
	for _, v := range m.Bins {
		aw.fprintf("%s%s", binsDelim, v)
	}
	aw.fprintf("\n")
	aw.fprintf("%s%s\n", commitDelim, m.CommitUUID)
	aw.fprintf("%s%s\n", namespaceDelim, m.Namespace)
	aw.fprintf("%s%t\n", headlessDelim, m.Headless)

	return aw.total()
}

// WriteToDisk is a convenience func, not intended to be unit tested.
func (m *ExecMeta) WriteToDisk(dir string) error {
	path := filepath.Join(dir, MetaFileName)
	buf := &bytes.Buffer{}
	if _, err := m.WriteTo(buf); err != nil {
		return err
	}
	if err := writeFile(path, buf.Bytes()); err != nil {
		return err
	}
	return nil
}

type accumulatingWrite struct {
	w   io.Writer
	n   int64
	err error
}

func newAccumulatingWrite(w io.Writer) *accumulatingWrite {
	return &accumulatingWrite{
		w: w,
	}
}

func (aw *accumulatingWrite) fprintf(format string, as ...interface{}) {
	if aw.err != nil {
		return
	}
	n, err := fmt.Fprintf(aw.w, format, as...)
	if err != nil {
		aw.err = err
		return
	}
	aw.n += int64(n)
}

func (aw accumulatingWrite) total() (int64, error) {
	return aw.n, aw.err
}

func IsMetaFile(fileContents []byte) bool {
	return strings.Contains(string(fileContents), envDelim)
}

func readFile(filePath string) ([]byte, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadFile %s failed: %w", filePath, err)
	}
	return b, nil
}

func writeFile(filePath string, data []byte) error {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("os.OpenFile %s failed: %w", filePath, err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("file.Write %s failed: %w", filePath, err)
	}
	return nil
}
