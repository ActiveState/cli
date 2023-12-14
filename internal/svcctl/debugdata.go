package svcctl

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

type execKind string

const (
	startSvc execKind = "Start"
	stopSvc  execKind = "Stop"
)

type debugData struct {
	argText string

	sockInfo    *fileInfo
	sockDirInfo *fileInfo
	sockDirList []string

	execStart time.Time
	execDur   time.Duration
	execKind  execKind

	waitStart    time.Time
	waitAttempts []*waitAttempt
	waitDur      time.Duration
}

func newDebugData(ipComm IPCommunicator, kind execKind, argText string) *debugData {
	sock := ipComm.SockPath().String()
	sockDir := filepath.Dir(sock)

	return &debugData{
		argText:     argText,
		sockInfo:    newFileInfo(sock),
		sockDirInfo: newFileInfo(sockDir),
		sockDirList: fileutils.ListDirSimple(sockDir, false),
		execStart:   time.Now(),
		execKind:    kind,
	}
}

func (d *debugData) Error() string {
	var sockDirList string
	for _, entry := range d.sockDirList {
		sockDirList += "\n  " + entry
	}
	if sockDirList == "" {
		sockDirList = "No entries found"
	}

	var attemptsMsg string
	for _, wa := range d.waitAttempts {
		attemptsMsg += "\n  " + wa.LogString()
	}

	return fmt.Sprintf(strings.TrimSpace(`
Arg Text      : %s
Sock Info     : %s
Sock Dir Info : %s
Sock Dir List : %s
%s Start    : %s
%s Duration : %s
Wait Start    : %s
Wait Duration : %s
Wait Log: %s
`),
		d.argText,
		strings.ReplaceAll(d.sockInfo.LogString(), "\n", "\n  "),
		strings.ReplaceAll(d.sockDirInfo.LogString(), "\n", "\n  "),
		sockDirList,
		d.execKind, d.execStart,
		d.execKind, d.execDur,
		d.waitStart,
		d.waitDur,
		attemptsMsg,
	)
}

func (d *debugData) startWait() {
	d.execDur = time.Since(d.execStart)
	logging.Debug("Exec time before wait was %v", d.execDur)
	d.waitStart = time.Now()
}

func (d *debugData) addWaitAttempt(start time.Time, iter int, timeout time.Duration) {
	attempt := &waitAttempt{
		waitStart: d.waitStart,
		start:     start,
		iter:      iter,
		timeout:   timeout,
	}
	d.waitAttempts = append(d.waitAttempts, attempt)
	logging.Debug("%s", attempt)
}

func (d *debugData) stopWait() {
	d.waitDur = time.Since(d.waitStart)
	logging.Debug("Wait duration: %s", d.waitDur)
}

type fileInfo struct {
	path string
	fs.FileInfo
	osFIErr error
}

func newFileInfo(path string) *fileInfo {
	fi := &fileInfo{path: path}
	fi.FileInfo, fi.osFIErr = os.Stat(path)
	return fi
}

func (f *fileInfo) LogString() string {
	if f.osFIErr != nil {
		return fmt.Sprintf("Path: %s Error: %s", f.path, f.osFIErr)
	}
	return fmt.Sprintf("Path: %s Size: %d Mode: %s ModTime: %s", f.path, f.Size(), f.Mode(), f.ModTime())
}

type waitAttempt struct {
	waitStart time.Time
	start     time.Time
	iter      int
	timeout   time.Duration
}

func (a *waitAttempt) String() string {
	return fmt.Sprintf("Attempt %2d at %10s with timeout %v",
		a.iter, a.start.Sub(a.waitStart).Round(time.Microsecond), a.timeout)
}

func (a *waitAttempt) LogString() string {
	return fmt.Sprintf("%2d: %10s/%v", a.iter, a.start.Sub(a.waitStart).Round(time.Microsecond), a.timeout)
}
