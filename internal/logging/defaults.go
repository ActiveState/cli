//go:build !test
// +build !test

package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation/storage"
)

// datadir is the base directory at which the log is saved
var datadir string

var timestamp int64

const FileNameSuffix = ".log"

// Logger describes a logging function, like Debug, Error, Warning, etc.
type Logger func(msg string, args ...interface{})

type safeBool struct {
	mu sync.Mutex
	v  bool
}

func (s *safeBool) value() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.v
}

func (s *safeBool) setValue(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.v = v
}

func FileName() string {
	return FileNameFor(os.Getpid())
}

func FileNameFor(pid int) string {
	return FileNameForCmd(FileNamePrefix(), pid)
}

func FileNameForCmd(cmd string, pid int) string {
	if cmd == constants.StateInstallerCmd {
		return fmt.Sprintf("%s-%d%s", cmd, pid, FileNameSuffix)
	}
	return fmt.Sprintf("%s-%d-%d%s", cmd, pid, timestamp, FileNameSuffix)
}

func FileNamePrefix() string {
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}
	exe = filepath.Base(exe)
	return strings.TrimSuffix(exe, filepath.Ext(exe))
}

func FilePath() string {
	return FilePathFor(FileName())
}

func FilePathFor(filename string) string {
	return filepath.Join(datadir, "logs", filename)
}

func FilePathForCmd(cmd string, pid int) string {
	return FilePathFor(FileNameForCmd(cmd, pid))
}

func init() {
	defer func() { handlePanics(recover()) }()

	// Set up datadir
	var err error
	datadir, err = storage.AppDataPath()
	if err != nil {
		log.SetOutput(os.Stderr)
		Error("Could not detect AppData dir: %v", err)
		return
	}

	// Set up handler
	timestamp = time.Now().UnixNano()
	handler := newFileHandler()
	SetHandler(handler)
	handler.SetVerbose(os.Getenv("VERBOSE") != "")
	log.SetOutput(&writer{})
}
