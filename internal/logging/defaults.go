// +build !test

package logging

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
)

// datadir is the base directory at which the log is saved
var datadir string

var timestamp int64

// CurrentCmd holds the value of the current command being invoked
// it's a quick hack to allow us to log the command to rollbar without risking exposing sensitive info
var CurrentCmd string

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
	defer handlePanics(recover())
	timestamp = time.Now().UnixNano()
	handler := newFileHandler()
	SetHandler(handler)

	log.SetOutput(&writer{})

	// Clean up old log files
	var err error
	datadir, err = storage.AppDataPath()
	if err != nil {
		Error("Could not detect AppData dir: %v", err)
		return
	}

	files, err := ioutil.ReadDir(datadir)
	if err != nil && !os.IsNotExist(err) {
		Error("Could not scan config dir to clean up stale logs: %v", err)
		return
	}

	sort.Slice(files, func(i, j int) bool { return files[i].ModTime().After(files[j].ModTime()) })
	files = funk.Filter(files, func(f fs.FileInfo) bool {
		return f.ModTime().Before(time.Now().Add(-time.Hour))
	}).([]fs.FileInfo)

	c := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), FileNamePrefix()) && strings.HasSuffix(file.Name(), FileNameSuffix) {
			c = c + 1
			if c > 9 {
				if err := os.Remove(filepath.Join(datadir, file.Name())); err != nil {
					Error("Could not clean up old log: %s, error: %v", file.Name(), err)
				}
			}
		}
	}

	Debug("Args: %v", os.Args)
}
