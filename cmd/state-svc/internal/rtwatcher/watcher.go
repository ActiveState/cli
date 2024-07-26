package rtwatcher

import (
	"encoding/json"
	"errors"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/panics"
)

const defaultInterval = 1 * time.Minute
const CfgKey = "runtime-watchers"

type Watcher struct {
	an       analytics
	cfg      *config.Instance
	watching []entry
	stop     chan struct{}
	interval time.Duration
}

type analytics interface {
	EventWithSource(category, action, source string, dim ...*dimensions.Values)
}

func New(cfg *config.Instance, an analytics) *Watcher {
	w := &Watcher{an: an, stop: make(chan struct{}, 1), cfg: cfg, interval: defaultInterval}

	if watchersJson := w.cfg.GetString(CfgKey); watchersJson != "" {
		watchers := []entry{}
		err := json.Unmarshal([]byte(watchersJson), &watchers)
		if err != nil {
			multilog.Error("Could not unmarshal watchersL %s", errs.JoinMessage(err))
		} else {
			w.watching = watchers
		}
	}

	if v := os.Getenv(constants.HeartbeatIntervalEnvVarName); v != "" {
		vv, err := strconv.Atoi(v)
		if err != nil {
			logging.Warning("Invalid value for %s: %s", constants.HeartbeatIntervalEnvVarName, v)
		} else {
			w.interval = time.Duration(vv) * time.Millisecond
		}
	}

	go w.ticker(w.check)
	return w
}

func (w *Watcher) ticker(cb func()) {
	defer func() { panics.LogPanics(recover(), debug.Stack()) }()

	logging.Debug("Starting watcher ticker with interval %s", w.interval.String())
	ticker := time.NewTicker(w.interval)
	for {
		select {
		case <-ticker.C:
			cb()
		case <-w.stop:
			logging.Debug("Stopping watcher ticker")
			return
		}
	}
}

func (w *Watcher) check() {
	watching := w.watching[:0]
	for i := range w.watching {
		e := w.watching[i] // Must use index, because we are deleting indexes further down
		running, err := e.IsRunning()
		var errProcess *processError
		if err != nil && !errors.As(err, &errProcess) {
			multilog.Error("Could not check if runtime process is running: %s", errs.JoinMessage(err))
			// Don't return yet, the conditional below still needs to clear this entry
		}
		if !running {
			logging.Debug("Runtime process %d:%s is not running, removing from watcher", e.PID, e.Exec)
			continue
		}
		watching = append(watching, e)

		go w.RecordUsage(e)
	}
	w.watching = watching
}

func (w *Watcher) RecordUsage(e entry) {
	logging.Debug("Recording usage of %s (%d)", e.Exec, e.PID)
	w.an.EventWithSource(anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat, e.Source, e.Dims)
}

func (w *Watcher) GetProcessesInUse(execDir string) []entry {
	inUse := make([]entry, 0)

	execDir = strings.ToLower(execDir) // match case-insensitively
	for _, proc := range w.watching {
		if !strings.Contains(strings.ToLower(proc.Exec), execDir) {
			continue
		}
		isRunning, err := proc.IsRunning()
		var errProcess *processError
		if err != nil && !errors.As(err, &errProcess) {
			multilog.Error("Could not check if runtime process is running: %s", errs.JoinMessage(err))
			// Any errors should not affect fetching which processes are currently in use. We just won't
			// include this one in the list.
		}
		if !isRunning {
			logging.Debug("Runtime process %d:%s is not running", proc.PID, proc.Exec)
			continue
		}
		inUse = append(inUse, proc) // append a copy
	}

	return inUse
}

func (w *Watcher) Close() error {
	logging.Debug("Closing runtime watcher")

	close(w.stop)

	if len(w.watching) > 0 {
		watchingJson, err := json.Marshal(w.watching)
		if err != nil {
			return errs.Wrap(err, "Could not marshal watchers")
		}
		return w.cfg.Set(CfgKey, watchingJson)
	}

	return nil
}

func (w *Watcher) Watch(pid int, exec, source string, dims *dimensions.Values) {
	logging.Debug("Watching %s (%d)", exec, pid)
	dims.Sequence = ptr.To(-1) // sequence is meaningless for heartbeat events
	e := entry{pid, exec, source, dims}
	w.watching = append(w.watching, e)
	go w.RecordUsage(e) // initial event
}
