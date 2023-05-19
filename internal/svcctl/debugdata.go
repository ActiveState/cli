package svcctl

import (
	"fmt"
	"strings"
	"time"
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
		"Attempt %02d at %12s with timeout %v",
		a.iter, a.start.Sub(a.waitStart), a.timeout,
	)
}

func (a *waitAttempt) LogString() string {
	return fmt.Sprintf(
		"%02d at %12s, wait %v",
		a.iter, a.start.Sub(a.waitStart), a.timeout,
	)
}

type waitDebugData struct {
	execStart    time.Time
	execDur      time.Duration
	execKind     execKind
	waitStart    time.Time
	waitAttempts []*waitAttempt
	waitDur      time.Duration
}

func newWaitDebugData(kind execKind) *waitDebugData {
	return &waitDebugData{
		execStart: time.Now(),
		execKind:  kind,
	}
}

func (d *waitDebugData) Error() string {
	var attemptsMsg string
	sep := "  "
	for _, wa := range d.waitAttempts {
		attemptsMsg += sep + wa.LogString()
		sep = "\n  "
	}
	return fmt.Sprintf(strings.TrimSpace(`
%s Start    : %s
%s Duration : %s
Wait Start    : %s
Wait Duration : %s
Wait Log:
%s
`),
		d.execKind, d.execStart,
		d.execKind, d.execDur,
		d.waitStart,
		d.waitDur,
		attemptsMsg)
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
