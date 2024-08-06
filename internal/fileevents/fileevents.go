package fileevents

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileevents/watcher"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/project"
)

const eventName = "file-changed"

type FileEvents struct {
	watcher *watcher.Watcher
	pj      *project.Project
}

func New(pj *project.Project) (*FileEvents, error) {
	fe := &FileEvents{pj: pj}
	var err error

	fe.watcher, err = watcher.New()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create watcher")
	}

	events := pj.Events()
	for _, event := range events {
		if event.Name() != eventName {
			continue
		}
		scope, err := event.Scope()
		if err != nil {
			return nil, locale.WrapError(err, "err_fileevent_scope", "Could not get scope for event: {{.V0}}.", event.Name())
		}
		for _, s := range scope {
			if err := fe.watcher.Add(filepath.Join(filepath.Dir(pj.Source().Path()), s)); err != nil {
				return nil, locale.WrapError(err, "err_fileevent_addwatcher", "Could not watch for file events on {{.V0}}.", s)
			}
		}
	}

	if err := fe.watcher.OnEvent(fe.onEvent); err != nil {
		return nil, errs.Wrap(err, "Could not set event handler")
	}
	return fe, nil
}

func (fe *FileEvents) onEvent(affectedFilepath string, log logging.Logger) error {
	logging.Debug("fileevent onEvent: %s", affectedFilepath)
	for _, event := range fe.pj.Events() {
		if event.Name() != eventName {
			continue
		}

		eventPaths, err := event.Scope()
		if err != nil {
			return locale.WrapError(err, "err_fileevent_scope", "Could not get scope for event: {{.V0}}.", event.Name())
		}

		projectPath := filepath.Clean(filepath.Dir(fe.pj.Source().Path()))
		affectedFilepath = filepath.Clean(affectedFilepath)

		logging.Debug("checking %s against %v", affectedFilepath, eventPaths)

		match := false
		for _, eventPath := range eventPaths {
			absoluteEventPath := filepath.Join(projectPath, eventPath)
			absoluteEventPath = filepath.Clean(absoluteEventPath)

			pathsEqual, err := fileutils.PathsEqual(affectedFilepath, absoluteEventPath)
			if err != nil {
				return locale.NewError("err_fileevent_equal", "Could not check if paths {{.V0}} and {{.V1}} are equal: {{.V2}}.", affectedFilepath, absoluteEventPath, err.Error())
			}
			if pathsEqual {
				match = true
				break
			}

			pathContainsParent, err := fileutils.PathContainsParent(affectedFilepath, absoluteEventPath)
			if err != nil {
				return locale.NewError("err_fileevent_equal", "Could not check if {{.V0}} is a child of {{.V1}}: {{.V2}}.", affectedFilepath, absoluteEventPath, err.Error())
			}

			if pathContainsParent {
				match = true
				break
			}
		}
		if !match {
			continue
		}

		value, err := event.Value()
		if err != nil {
			return locale.WrapError(err, "err_fileevent_scope", "Could not get value for event: {{.V0}}.", event.Name())
		}

		logger := func(msg string, args ...interface{}) {
			log(fmt.Sprintf("%s: ", value)+msg, args...)
		}
		err = runScript(value, logger)
		if err != nil {
			return locale.NewError("err_fileevent_cmd", "Could not run the script `{{.V0}}`. Please ensure its name is valid and you can run `state run {{.V0}}`.", value)
		}
	}
	return nil
}

func (fe *FileEvents) Close() {
	logging.Debug("Closing fileevents")
	fe.watcher.Close()
}

func runScript(name string, log logging.Logger) error {
	exe, err := os.Executable()
	if err != nil {
		return locale.NewError("err_os_exe", "Could not retrieve path of current executable.")
	}

	log(locale.Tl("running_script", "Running Script"))
	cmd := exec.Command(exe, "run", name)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errs.Wrap(err, "Could not pipe stderr")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errs.Wrap(err, "Could not pipe stdout")
	}

	go captureStd(stderr, log)
	go captureStd(stdout, log)

	if err := cmd.Start(); err != nil {
		return locale.WrapError(err, "err_fileevent_cmd_start", "Could not run script `{{.V0}}`. Error: {{.V1}}.", name, err.Error())
	}
	if err := cmd.Wait(); err != nil {
		return locale.WrapError(err, "err_fileevent_cmd_start", "Error happened while running script `{{.V0}}`. Error: {{.V1}}.", name, err.Error())
	}

	log(locale.Tl("script_finished", "Script Finished"))

	return nil
}

func captureStd(reader io.ReadCloser, log logging.Logger) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		log(scanner.Text())
	}
}
