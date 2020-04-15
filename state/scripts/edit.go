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
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/fsnotify/fsnotify"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
)

// The default open command and editors based on platform
const (
	openCmdLin       = "xdg-open"
	openCmdMac       = "open"
	defaultEditorWin = "notepad.exe"
)

var (
	// FailWatcherRead indicates a failure reading from a watcher channel
	FailWatcherRead = failures.Type("edit.fail.watcherread")

	// FailWatcherInstance indicates a failure from the active watcher
	FailWatcherInstance = failures.Type("edit.fail.watcherinstance")

	// FailInvalidEditor indicates the EDITOR variable is not correctly set
	FailInvalidEditor = failures.Type("edit.fail.invalideditor")
)

// EditArgs captures values for any arguments used with the edit command
var EditArgs struct {
	Name string
}

// EditFlags captures values for any of the flags used with the edit command
var EditFlags struct {
	Expand bool
}

// EditCommand represents an edit sub-comand
var EditCommand = &commands.Command{
	Name:        "edit",
	Description: "edit_description",
	Run:         ExecuteEdit,
	Arguments: []*commands.Argument{
		{
			Name:        "edit_script_cmd_name_arg",
			Description: "edit_script_cmd_name_arg_description",
			Variable:    &EditArgs.Name,
			Required:    true,
		},
	},
	Flags: []*commands.Flag{
		{
			Name:        "expand",
			Shorthand:   "e",
			Description: "edit_script_cmd_expand_flag",
			Type:        commands.TypeBool,
			BoolVar:     &EditFlags.Expand,
		},
	},
}

// ExecuteEdit runs the edit command
func ExecuteEdit(cmd *cobra.Command, args []string) {
	script := project.Get().ScriptByName(EditArgs.Name)
	if script == nil {
		print.Line(locale.Tr("edit_scripts_no_name", EditArgs.Name))
		return
	}

	fail := editScript(script)
	if fail != nil {
		failures.Handle(fail, locale.T("error_edit_script"))
		return
	}
}

func editScript(script *project.Script) *failures.Failure {
	scriptFile, fail := createScriptFile(script)
	if fail != nil {
		return fail
	}
	defer scriptFile.Clean()

	watcher, fail := newScriptWatcher(scriptFile)
	if fail != nil {
		return fail
	}
	defer watcher.close()

	fail = openEditor(scriptFile.Filename())
	if fail != nil {
		return fail
	}

	return start(watcher)
}

func createScriptFile(script *project.Script) (*scriptfile.ScriptFile, *failures.Failure) {
	scriptBlock := script.Raw()
	if EditFlags.Expand {
		scriptBlock = script.Value()
	}

	return scriptfile.NewAsSource(script.LanguageSafe(), script.Name(), scriptBlock)
}

func openEditor(filename string) *failures.Failure {
	editorCmd, fail := getOpenCmd()
	if fail != nil {
		return fail
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
			return failures.FailCmd.Wrap(err)
		}
	} else {
		err := subCmd.Run()
		if err != nil {
			return failures.FailCmd.Wrap(err)
		}
	}

	return nil
}

func getOpenCmd() (string, *failures.Failure) {
	editor := os.Getenv("EDITOR")
	if editor != "" {
		return verifyEditor(editor)
	}

	switch runtime.GOOS {
	case "linux":
		_, err := exec.LookPath(openCmdLin)
		if err != nil {
			return "", failures.FailNotFound.New("error_open_not_installed_lin", openCmdLin)
		}
		return openCmdLin, nil
	case "darwin":
		return openCmdMac, nil
	case "windows":
		return defaultEditorWin, nil
	default:
		return "", failures.FailRuntime.New("error_edit_unrecognized_platform", runtime.GOOS)
	}
}

func verifyEditor(editor string) (string, *failures.Failure) {
	if strings.Contains(editor, string(os.PathSeparator)) {
		return verifyPathEditor(editor)
	}

	_, err := exec.LookPath(editor)
	if err != nil {
		return "", FailInvalidEditor.Wrap(err)
	}

	return editor, nil
}

func verifyPathEditor(editor string) (string, *failures.Failure) {
	if runtime.GOOS == "windows" && filepath.Ext(editor) == "" {
		return "", FailInvalidEditor.New("error_edit_windows_invalid_editor")
	}

	_, err := os.Stat(editor)
	if err != nil {
		return "", FailInvalidEditor.Wrap(err)
	}

	return editor, nil
}

func start(sw *scriptWatcher) *failures.Failure {
	print.Line("Watching file changes at: %s", sw.scriptFile.Filename())
	if strings.ToLower(os.Getenv(constants.NonInteractive)) == "true" {
		startNoninteractive(sw)
	} else {
		fail := startInteractive(sw)
		if fail != nil {
			return fail
		}
	}

	return nil
}

func startNoninteractive(sw *scriptWatcher) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		sig := <-c
		logging.Debug(fmt.Sprintf("Detected: %s handling any failures encountered while watching file", sig))
		defer func() {
			sw.done <- true
			sw.close()
			sw.scriptFile.Clean()
			Command.Exiter(1)
		}()
		select {
		case fail := <-sw.fails:
			failures.Handle(fail, locale.T("error_edit_watcher_fail"))
		default:
			// Do nothing and let defer take over
		}
	}()
	sw.run()
}

func startInteractive(sw *scriptWatcher) *failures.Failure {
	go sw.run()

	prompter := prompt.New()
	for {
		doneEditing, fail := prompter.Confirm(locale.T("prompt_done_editing"), true)
		if fail != nil {
			return fail
		}
		if doneEditing {
			sw.done <- true
			break
		}
	}

	select {
	case fail := <-sw.fails:
		return fail
	default:
		return nil
	}
}

type scriptWatcher struct {
	watcher    *fsnotify.Watcher
	scriptFile *scriptfile.ScriptFile
	done       chan bool
	fails      chan *failures.Failure
}

func newScriptWatcher(scriptFile *scriptfile.ScriptFile) (*scriptWatcher, *failures.Failure) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, failures.FailOS.Wrap(err)
	}

	err = watcher.Add(scriptFile.Filename())
	if err != nil {
		return nil, failures.FailOS.Wrap(err)
	}

	return &scriptWatcher{
		watcher:    watcher,
		scriptFile: scriptFile,
		done:       make(chan bool),
		fails:      make(chan *failures.Failure),
	}, nil
}

func (sw *scriptWatcher) run() {
	for {
		select {
		case <-sw.done:
			return
		case event, ok := <-sw.watcher.Events:
			if !ok {
				sw.fails <- FailWatcherRead.New("error_edit_watcher_channel_closed")
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				fail := updateProjectFile(sw.scriptFile)
				if fail != nil {
					sw.fails <- fail
					return
				}
				// To ensure confirm dialog and update message are not on the same line
				print.Line("")
				print.Line(locale.T("edit_scripts_project_file_saved"))
			}
		case err, ok := <-sw.watcher.Errors:
			if !ok {
				sw.fails <- FailWatcherRead.New("error_edit_watcher_channel_closed")
				return
			}
			sw.fails <- FailWatcherInstance.Wrap(err)
			return
		}
	}
}

func (sw *scriptWatcher) close() {
	sw.watcher.Close()
	close(sw.done)
	close(sw.fails)
}

func updateProjectFile(scriptFile *scriptfile.ScriptFile) *failures.Failure {
	updatedScript, fail := fileutils.ReadFile(scriptFile.Filename())
	if fail != nil {
		return fail
	}

	projectFile := project.Get().Source()
	i := constraints.MostSpecificUnconstrained(EditArgs.Name, projectFile.Scripts.AsConstrainedEntities())
	if i < 0 { // no script found
		return nil
	}
	projectFile.Scripts[i].Value = string(updatedScript)
	return projectFile.Save()

}
