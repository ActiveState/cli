package scripts

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/fsnotify/fsnotify"
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
	output output.Outputer
}

// NewEdit creates a new Edit runner
func NewEdit(output output.Outputer) *Edit {
	return &Edit{output}
}

func (e *Edit) Run(pj *project.Project, params *EditParams) error {
	script := pj.ScriptByName(params.Name)
	if script == nil {
		return locale.NewInputError("edit_scripts_no_name", "Could not find script with the given name {{.V0}}", params.Name)
	}

	err := e.editScript(script, params)
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

	err = openEditor(scriptFile.Filename())
	if err != nil {
		return locale.WrapError(
			err, "error_edit_open_scriptfile",
			"Failed to open {{.V0}} in editor.", scriptFile.Filename())
	}

	return start(watcher, params.Name, e.output)
}

func createScriptFile(script *project.Script, expand bool) (*scriptfile.ScriptFile, error) {
	scriptBlock := script.Raw()
	if expand {
		scriptBlock = script.Value()
	}

	f, fail := scriptfile.NewAsSource(script.LanguageSafe(), script.Name(), scriptBlock)
	if fail != nil {
		return f, errs.Wrap(fail, "Failed to create script file")
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

func (sw *scriptWatcher) run(scriptName string, outputer output.Outputer) {
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
			if event.Op&fsnotify.Write == fsnotify.Write {
				err := updateProjectFile(sw.scriptFile, scriptName)
				if err != nil {
					sw.errs <- err
					return
				}
				// To ensure confirm dialog and update message are not on the same line
				outputer.Print("")
				outputer.Print(locale.T("edit_scripts_project_file_saved"))
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

func openEditor(filename string) error {
	editorCmd, err := getOpenCmd()
	if err != nil {
		return err
	}

	subCmd := exec.Command(editorCmd, filename)

	// Command line editors like vim will detect if the input/output
	// is not from a proper terminal. Hence we have to redirect here
	subCmd.Stdin = os.Stdin
	subCmd.Stdout = os.Stdout
	subCmd.Stderr = os.Stderr

	if runtime.GOOS == "windows" && strings.Contains(editorCmd, defaultEditorWin) {
		err := subCmd.Start()
		if err != nil {
			return errs.Wrap(err, "Failed to start editor command.")
		}
	} else {
		err := subCmd.Run()
		if err != nil {
			return errs.Wrap(err, "Failed to run editor command.")
		}
	}

	return nil
}

func getOpenCmd() (string, error) {
	editor := os.Getenv("EDITOR")
	if editor != "" {
		return verifyEditor(editor)
	}

	switch runtime.GOOS {
	case "linux":
		_, err := exec.LookPath(openCmdLin)
		if err != nil {
			return "", locale.NewError(
				"error_open_not_installed_lin",
				"Please install '{{.V0}}' to edit scripts.",
				openCmdLin)
		}
		return openCmdLin, nil
	case "darwin":
		return openCmdMac, nil
	case "windows":
		return defaultEditorWin, nil
	default:
		return "", locale.NewError(
			"error_edit_unrecognized_platform",
			"Could not open script file on this platform {{.V0}}",
			runtime.GOOS)
	}
}

func verifyEditor(editor string) (string, error) {
	if strings.Contains(editor, string(os.PathSeparator)) {
		return verifyPathEditor(editor)
	}

	_, err := exec.LookPath(editor)
	if err != nil {
		return "", errs.Wrap(err, "Failed to find a suite-able editor.")
	}

	return editor, nil
}

func verifyPathEditor(editor string) (string, error) {
	if runtime.GOOS == "windows" && filepath.Ext(editor) == "" {
		return "", locale.NewError(
			"error_edit_windows_invalid_editor",
			"Editor path must contain a file extension")
	}

	_, err := os.Stat(editor)
	if err != nil {
		return "", errs.Wrap(err, "Failed to find editor %s on file system.", editor)
	}

	return editor, nil
}

func start(sw *scriptWatcher, scriptName string, output output.Outputer) (err error) {
	output.Print(fmt.Sprintf("Watching file changes at: %s", sw.scriptFile.Filename()))
	if strings.ToLower(os.Getenv(constants.NonInteractive)) == "true" {
		err = startNoninteractive(sw, scriptName, output)
	} else {
		err = startInteractive(sw, scriptName, output)
	}
	return err
}

func startNoninteractive(sw *scriptWatcher, scriptName string, output output.Outputer) error {
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
	sw.run(scriptName, output)

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

func startInteractive(sw *scriptWatcher, scriptName string, output output.Outputer) error {
	go sw.run(scriptName, output)

	prompter := prompt.New()
	for {
		doneEditing, fail := prompter.Confirm(locale.T("prompt_done_editing"), true)
		if fail != nil {
			return errs.Wrap(fail, "Prompter returned with failure.")
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

func updateProjectFile(scriptFile *scriptfile.ScriptFile, name string) error {
	updatedScript, fail := fileutils.ReadFile(scriptFile.Filename())
	if fail != nil {
		return errs.Wrap(fail, "Failed to read script file %s.", scriptFile.Filename())
	}

	projectFile := projectfile.Get()
	i := constraints.MostSpecificUnconstrained(name, projectFile.Scripts.AsConstrainedEntities())
	if i < 0 { // no script found
		return nil
	}
	projectFile.Scripts[i].Value = string(updatedScript)

	fail = projectFile.Save()
	if fail != nil {
		return errs.Wrap(fail, "Failed to save project file.")
	}
	return nil
}
