package scripts

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"

	"github.com/fsnotify/fsnotify"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// The default open command and editors based on platform
const (
	openCmdLin       = "xdg-open"
	openCmdMac       = "open"
	defaultEditorWin = "notepad.exe"
)

// EditParams stores command line parameters for script edit commands
type EditParams struct {
	Name   string
	Expand bool
}

// Edit represents the runner for `state script edit`
type Edit struct {
	project  *project.Project
	output   output.Outputer
	prompter prompt.Prompter
	cfg      projectfile.ConfigGetter
}

// NewEdit creates a new Edit runner
func NewEdit(prime primeable) *Edit {
	return &Edit{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
		prime.Config(),
	}
}

func (e *Edit) Run(params *EditParams) error {
	if e.project == nil {
		return rationalize.ErrNoProject
	}

	script, err := e.project.ScriptByName(params.Name)
	if err != nil {
		return errs.Wrap(err, "Could not get script")
	}
	if script == nil {
		return locale.NewInputError("edit_scripts_no_name", "Could not find script with the given name {{.V0}}", params.Name)
	}

	err = e.editScript(script, params)
	if err != nil {
		return locale.WrapError(err, "error_edit_script", "Failed to edit script.")
	}
	return nil
}

func (e *Edit) editScript(script *project.Script, params *EditParams) error {
	scriptFile, err := createScriptFile(script, params.Expand)
	if err != nil {
		return locale.WrapError(
			err, "error_edit_create_scriptfile",
			"Could not create script file.")
	}
	defer scriptFile.Clean()

	watcher, err := newScriptWatcher(scriptFile)
	if err != nil {
		return errs.Wrap(err, "Failed to initialize file watch.")
	}
	defer watcher.close()

	err = osutils.OpenEditor(scriptFile.Filename())
	if err != nil {
		return locale.WrapError(
			err, "error_edit_open_scriptfile",
			"Failed to open script file in editor.")
	}

	return start(e.prompter, watcher, params.Name, e.output, e.cfg, e.project)
}

func createScriptFile(script *project.Script, expand bool) (*scriptfile.ScriptFile, error) {
	scriptBlock := script.Raw()
	if expand {
		var err error
		scriptBlock, err = script.Value()
		if err != nil {
			return nil, errs.Wrap(err, "Could not get script value")
		}
	}

	languages := script.LanguageSafe()
	if len(languages) == 0 {
		languages = project.DefaultScriptLanguage()
	}

	f, err := scriptfile.NewAsSource(languages[0], script.Name(), scriptBlock)
	if err != nil {
		return f, errs.Wrap(err, "Failed to create script file")
	}
	return f, nil
}

type scriptWatcher struct {
	watcher    *fsnotify.Watcher
	scriptFile *scriptfile.ScriptFile
	done       chan bool
	errs       chan error
}

func newScriptWatcher(scriptFile *scriptfile.ScriptFile) (*scriptWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errs.Wrap(err, "failed to create file watcher")
	}

	err = watcher.Add(scriptFile.Filename())
	if err != nil {
		return nil, errs.Wrap(err, "failed to add %s to file watcher", scriptFile.Filename())
	}

	return &scriptWatcher{
		watcher:    watcher,
		scriptFile: scriptFile,
		done:       make(chan bool),
		errs:       make(chan error),
	}, nil
}

func (sw *scriptWatcher) run(scriptName string, outputer output.Outputer, cfg projectfile.ConfigGetter, proj *project.Project) {
	for {
		select {
		case <-sw.done:
			return
		case event, ok := <-sw.watcher.Events:
			if !ok {
				sw.errs <- locale.NewError(
					"error_edit_watcher_channel_closed",
					"Encountered error watching scriptfile. Please restart edit command.",
				)
				return
			}
			// Some editors do not set WRITE events when a file is modified. Instead they send a REMOVE event
			// followed by a CREATE event. The script file already exists at this point so we capture the
			// CREATE event as a WRITE event.
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				err := updateProjectFile(cfg, proj, sw.scriptFile, scriptName)
				if err != nil {
					sw.errs <- errs.Wrap(err, "Failed to write project file.")
					return
				}
				outputer.Notice(locale.T("edit_scripts_project_file_saved"))
			}
		case err, ok := <-sw.watcher.Errors:
			if !ok {
				sw.errs <- locale.NewError(
					"error_edit_watcher_channel_closed",
					"Encountered error watching scriptfile. Please restart edit command.")
				return
			}
			sw.errs <- errs.Wrap(err, "File watcher reported error.")
			return
		}
	}
}

func (sw *scriptWatcher) close() {
	sw.watcher.Close()
	close(sw.done)
	close(sw.errs)
}

func start(prompt prompt.Prompter, sw *scriptWatcher, scriptName string, output output.Outputer, cfg projectfile.ConfigGetter, proj *project.Project) (err error) {
	output.Notice(locale.Tr("script_watcher_watch_file", sw.scriptFile.Filename()))
	if prompt.IsInteractive() {
		return startInteractive(sw, scriptName, output, cfg, proj, prompt)
	}
	return startNoninteractive(sw, scriptName, output, cfg, proj)
}

func startNoninteractive(sw *scriptWatcher, scriptName string, output output.Outputer, cfg projectfile.ConfigGetter, proj *project.Project) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	errC := make(chan error)
	defer close(errC)
	go func() {
		sig := <-c
		logging.Debug(fmt.Sprintf("Detected: %s handling any failures encountered while watching file", sig))
		var err error
		defer func() {
			// signal the process that we are done
			sw.done <- true
			errC <- err
		}()
		select {
		case err = <-sw.errs:
		default:
			// Do nothing and let defer take over
		}
	}()
	sw.run(scriptName, output, cfg, proj)

	err := <-errC

	// clean-up
	sw.scriptFile.Clean()

	if err != nil {
		return locale.WrapError(
			err, "error_edit_watcher_fail",
			"An error occurred while watching for file changes.  Your changes may not be saved.")
	}
	return nil
}

func startInteractive(sw *scriptWatcher, scriptName string, output output.Outputer, cfg projectfile.ConfigGetter, proj *project.Project, prompt prompt.Prompter) error {
	go sw.run(scriptName, output, cfg, proj)

	for {
		doneEditing, err := prompt.Confirm("", locale.T("prompt_done_editing"), ptr.To(true), nil)
		if err != nil {
			return errs.Wrap(err, "Not confirmed")
		}
		if doneEditing {
			sw.done <- true
			break
		}
	}

	select {
	case err := <-sw.errs:
		return err
	default:
		return nil
	}
}

func updateProjectFile(cfg projectfile.ConfigGetter, pj *project.Project, scriptFile *scriptfile.ScriptFile, name string) error {
	updatedScript, err := fileutils.ReadFile(scriptFile.Filename())
	if err != nil {
		return errs.Wrap(err, "Failed to read script file %s.", scriptFile.Filename())
	}

	pjf := pj.Source()
	script, err := pj.ScriptByName(name)
	if err != nil {
		return errs.Wrap(err, "Could not get script")
	}
	if script == nil {
		return locale.NewError("err_update_script_cannot_find", "Could not find the source script to update.")
	}

	idx := -1
	for i, s := range pjf.Scripts {
		if reflect.DeepEqual(s, *script.SourceScript()) {
			idx = i
			break
		}
	}
	if idx == -1 {
		return locale.NewError("err_update_script_cannot_find", "Could not find the source script to update.")
	}

	pjf.Scripts[idx].Value = string(updatedScript)

	err = pjf.Save(cfg)
	if err != nil {
		return errs.Wrap(err, "Failed to save project file.")
	}
	return nil
}
