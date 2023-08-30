package svcctl

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/ipc"
)

type execKind string

const (
	execSvc execKind = "Exec"
	stopSvc execKind = "Stop"
)

type waitAttempt struct {
	waitStart time.Time
	start     time.Time
	iter      int
	timeout   time.Duration
}

func newWaitAttempt(waitStart, start time.Time, iter int, timeout time.Duration) *waitAttempt {
	return &waitAttempt{
		waitStart: waitStart,
		start:     start,
		iter:      iter,
		timeout:   timeout,
	}
}

func (a *waitAttempt) String() string {
	return fmt.Sprintf(
		"Attempt %2d at %10s with timeout %v",
		a.iter, a.start.Sub(a.waitStart).Round(time.Microsecond), a.timeout,
	)
}

func (a *waitAttempt) LogString() string {
	return fmt.Sprintf(
		"%2d: %10s/%v",
		a.iter, a.start.Sub(a.waitStart).Round(time.Microsecond), a.timeout,
	)
}

type waitDebugData struct {
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

func newWaitDebugData(sp *ipc.SockPath, kind execKind) *waitDebugData {
	sock := sp.String()
	sockDir := filepath.Dir(sock)

	return &waitDebugData{
		sockInfo:    newFileInfo(sock),
		sockDirInfo: newFileInfo(sockDir),
		sockDirList: fileutils.ListDirSimple(sockDir, false),
		execStart:   time.Now(),
		execKind:    kind,
	}
}

func (d *waitDebugData) Error() string {
	var sockDirList string
	sep := "    "
	for _, entry := range d.sockDirList {
		sockDirList += sep + entry
		sep = "\n"
	}
	if sockDirList == "" {
		sockDirList = "No entries found"
	}

	var attemptsMsg string
	sep = "  "
	for _, wa := range d.waitAttempts {
		attemptsMsg += sep + wa.LogString()
		sep = "\n  "
	}

	return fmt.Sprintf(strings.TrimSpace(`
Sock Info     : %s
Sock Dir Info : %s
Sock Dir List : %s
%s Start    : %s
%s Duration : %s
Wait Start    : %s
Wait Duration : %s
Wait Log:
%s
`),
		strings.ReplaceAll(d.sockInfo.LogString(), "\n", "\n    "),
		strings.ReplaceAll(d.sockDirInfo.LogString(), "\n", "\n    "),
		sockDirList,
		d.execKind, d.execStart,
		d.execKind, d.execDur,
		d.waitStart,
		d.waitDur,
		attemptsMsg,
	)
}

func (d *waitDebugData) stampExec() {
	if d.execStart.IsZero() {
		return
	}
	d.execDur = time.Since(d.execStart)
}

func (d *waitDebugData) startWait() {
	d.waitStart = time.Now()
}

func (d *waitDebugData) addAttempts(as ...*waitAttempt) {
	d.waitAttempts = append(d.waitAttempts, as...)
}

func (d *waitDebugData) stampWait() {
	if d.waitStart.IsZero() {
		return
	}
	d.waitDur = time.Since(d.waitStart)
}

type fileInfo struct {
	path string
	fs.FileInfo
	osFIErr error
}

func newFileInfo(path string) *fileInfo {
	fi := &fileInfo{
		path: path,
	}

	fi.FileInfo, fi.osFIErr = os.Stat(path)

	return fi
}

func (f *fileInfo) LogString() string {
	if f.osFIErr != nil {
		return fmt.Sprintf("%s, error: %s", f.path, f.osFIErr)
	}

	// name
	// size, mode, mod time
	return fmt.Sprintf(strings.TrimSpace(`
%s
%d, %s, %s
`),
		f.path,
		f.Size(), f.Mode(), f.ModTime(),
	)
}
