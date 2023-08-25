package rtwatcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"time"

	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cast"
)

const CfgKey = "runtime-watchers"
const LastUsedCfgKey = "runtime-last-used"

type Watcher struct {
	an       analytics
	cfg      *config.Instance
	watching []entry
	stop     chan struct{}
	interval time.Duration
}

type analytics interface {
	Event(category string, action string, dim ...*dimensions.Values)
}

func New(cfg *config.Instance, an analytics) *Watcher {
	w := &Watcher{an: an, stop: make(chan struct{}, 1), cfg: cfg, interval: constants.RuntimeHeartbeatInterval}

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
		if err != nil {
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
	w.an.Event(anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat, e.Dims)
	w.UpdateLastUsed(e)
}

func (w *Watcher) UpdateLastUsed(e entry) {
	if stateExec, err := installation.StateExec(); err == nil {
		if isStateExec, err := fileutils.PathsEqual(e.Exec, stateExec); err == nil && isStateExec {
			return // cannot infer which runtime is being used
		} else if err != nil {
			multilog.Error("Could not determine if %s is the State Tool executable: %s", e.Exec, errs.JoinMessage(err))
		}
	} else {
		multilog.Error("Could not determine the state tool executable: %s", errs.JoinMessage(err))
	}

	logging.Debug("Updating last usage of project that contains %s", e.Exec)
	localProjects := projectfile.GetProjectMapping(w.cfg)
	for namespace, checkouts := range localProjects {
		for _, checkout := range checkouts {
			proj, err := project.FromPath(checkout)
			if err != nil {
				logging.Error("Unable to get project %s from checkout: %v", checkout, err) // do not send to rollbar
				continue
			}
			projectTarget := target.NewProjectTarget(proj, nil, "")
			execDir := setup.ExecDir(projectTarget.Dir())
			logging.Debug("Looking at project %s located at %s whose executables are in %s", namespace, checkout, execDir)
			if inRuntime, err := fileutils.PathsEqual(filepath.Dir(e.Exec), execDir); err != nil || !inRuntime {
				if err != nil {
					multilog.Error("Unable to determine if this executable is in the project: %s", errs.JoinMessage(err))
				}
				continue
			}
			logging.Debug("Executable is in this runtime; updating 'last used' time")
			err = w.cfg.GetThenSet(
				LastUsedCfgKey,
				func(v interface{}) (interface{}, error) {
					lastUsed := cast.ToStringMap(v)
					lastUsed[execDir] = time.Now().Format(time.RFC3339)
					return lastUsed, nil
				})
			if err != nil {
				multilog.Error("Unable to update last used time: %s", errs.JoinMessage(err))
			}
			return
		}
	}
	logging.Debug("Unable to find project associated with %s to update last usage for", e.Exec)
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

func (w *Watcher) Watch(pid int, exec string, dims *dimensions.Values) {
	logging.Debug("Watching %s (%d)", exec, pid)
	dims.Sequence = ptr.To(-1) // sequence is meaningless for heartbeat events
	e := entry{pid, exec, dims}
	w.watching = append(w.watching, e)
	go w.RecordUsage(e) // initial event
}
