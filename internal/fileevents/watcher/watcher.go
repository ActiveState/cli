package watcher

import (
	"github.com/fsnotify/fsnotify"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rollbar"
)

type Closer func()

type OnEvent func(filepath string, log logging.Logger) error

func logInfo(msg string, args ...interface{}) {
	logging.Info("File-Event: "+msg, args...)
}

func logError(msg string, args ...interface{}) {
	logging.Error("File-Event: "+msg, args...)
	rollbar.Error("File-Event: "+msg, args...)
}

type Watcher struct {
	fswatcher *fsnotify.Watcher
	done      chan bool
	onEvent   *OnEvent
}

func New() (*Watcher, error) {
	w := &Watcher{}
	var err error
	w.fswatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create filesystem watcher")
	}

	w.done = make(chan bool)
	go func() {
		for {
			select {
			case <-w.done:
				return
			case event, ok := <-w.fswatcher.Events:
				if !ok || w.onEvent == nil {
					continue
				}

				if event.Op&fsnotify.Write != fsnotify.Write {
					logging.Debug(event.String() + ": Skip")
					continue
				}

				logInfo(event.String())

				if err := (*w.onEvent)(event.Name, logInfo); err != nil {
					logError(errs.Join(err, ", ").Error())
					continue
				}
			case err, ok := <-w.fswatcher.Errors:
				if !ok {
					return
				}
				logError(err.Error())
			}
		}
	}()

	return w, nil
}

func (w *Watcher) Add(filepath string) error {
	logInfo(locale.Tl("fileevent_watchig", "Watching {{.V0}}", filepath))
	if !fileutils.TargetExists(filepath) {
		return locale.NewInputError("err_fileevent_filenotexist", "Path does not exist: {{.V0}}.", filepath)
	}
	if err := w.fswatcher.Add(filepath); err != nil {
		return locale.WrapInputError(err, "err_fileevent_invalidpath", "Could not add filepath to filesystem watcher: {{.V0}}", filepath)
	}
	return nil
}

func (w *Watcher) OnEvent(cb OnEvent) error {
	if w.onEvent != nil {
		return errs.New("Already listening to events")
	}
	w.onEvent = &cb
	return nil
}

func (w *Watcher) Close() {
	logging.Debug("Closing watcher")
	w.fswatcher.Close()
	close(w.done)
	logging.Debug("Watcher closed")
}
