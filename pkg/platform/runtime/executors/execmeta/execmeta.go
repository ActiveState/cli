// Package execmeta models the executor meta data that is communicated from the
// state tool to executors via file. The meta file is stored alongside the
// executors and should be named "meta.as". It should be applied to the file
// file system when the runtime is manifested on disk.
//
// IMPORTANT: This package should have minimal dependencies as it will be
// imported by cmd/state-exec. The resulting compiled executable must remain as
// small as possible.
package execmeta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Target struct {
	CommitUUID string
	Namespace  string
	Dir        string
	Headless   bool
}

const (
	MetaFileName           = "meta.as"
	metaFileDetectionToken = `"SockPath":`
)

type ExecMeta struct {
	SockPath   string
	Env        []string
	Bins       map[string]string // map[alias]dest
	CommitUUID string
	Namespace  string
	Headless   bool
}

func New(sockPath string, env []string, t Target, bins map[string]string) *ExecMeta {
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
	if err := json.NewDecoder(r).Decode(&m); err != nil {
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

func (m *ExecMeta) Encode(w io.Writer) error {
	return json.NewEncoder(w).Encode(m)
}

// WriteToDisk is a convenience func, not intended to be unit tested.
func (m *ExecMeta) WriteToDisk(dir string) error {
	path := filepath.Join(dir, MetaFileName)
	buf := &bytes.Buffer{}
	if err := m.Encode(buf); err != nil {
		return err
	}
	if err := writeFile(path, buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func IsMetaFile(fileContents []byte) bool {
	return strings.Contains(string(fileContents), metaFileDetectionToken)
}

func readFile(filePath string) ([]byte, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile %s failed: %w", filePath, err)
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
