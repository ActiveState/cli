package scripts

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/language"

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
	defaultEditorLin = "vi"
	defaultEditorWin = "notepad"
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

	if EditArgs.Name == "" {
		failures.Handle(failures.FailUserInput.New("error_script_edit_undefined_name"), "")
		return
	}

	script := project.Get().ScriptByName(EditArgs.Name)
	if script == nil {
		fmt.Println(locale.Tr("edit_scripts_no_name", EditArgs.Name))
		return
	}

	fail := editScript(script)
	if fail != nil {
		failures.Handle(fail, "error_edit_script")
		os.Exit(1)
	}

}

func createScriptFile(script *project.Script) (*scriptfile.ScriptFile, *failures.Failure) {

	scriptBlock := script.Raw()
	if EditFlags.Expand {
		scriptBlock = script.Value()
	}

	return scriptfile.New(scriptLanguage(script), scriptBlock)

}

func scriptLanguage(script *project.Script) language.Language {
	if script.Language() == language.Unknown {
		return language.Sh
	}
	return script.Language()
}

func editScript(script *project.Script) *failures.Failure {

	scriptFile, fail := createScriptFile(script)
	if fail != nil {
		return fail
	}
	defer scriptFile.Clean()

	fail = openEditor(scriptFile.Filename())
	if fail != nil {
		return fail
	}

	prompter := prompt.New()
	for {
		yesDoneEditing, fail := prompter.Confirm(locale.T("prompt_done_editing"), true)
		if fail != nil {
			return fail
		}
		if yesDoneEditing {
			break
		}
	}

	// TODO: Ensure we can save with comments
	return updateProjectFile(scriptFile, script)

}

func openEditor(filename string) *failures.Failure {

	editorCmd, fail := getEditorCmd()
	if fail != nil {
		return fail
	}

	subCmd := exec.Command(editorCmd, filename)

	// Command line editors like vim will detect if the input/output
	// is not from a proper terminal. Hence we have to redirect here
	subCmd.Stdin = os.Stdin
	subCmd.Stdout = os.Stdout

	err := subCmd.Run()
	if err != nil {
		return failures.FailCmd.Wrap(err)
	}

	return nil

}

func getEditorCmd() (string, *failures.Failure) {

	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}
	fmt.Println(locale.T("edit_script_editor_not_set"))

	platform := runtime.GOOS
	switch platform {
	case "linux":
		_, err := exec.LookPath(openCmdLin)
		if err != nil {
			return defaultEditorLin, nil
		}
		return openCmdLin, nil
	case "darwin":
		return openCmdMac, nil
	case "windows":
		return defaultEditorWin, nil
	default:
		return "", failures.FailRuntime.New("error_edit_unrecognized_platform", platform)
	}

}

func updateProjectFile(scriptFile *scriptfile.ScriptFile, script *project.Script) *failures.Failure {

	scriptBytes, fail := fileutils.ReadFile(scriptFile.Filename())
	if fail != nil {
		return fail
	}
	// TODO: Find a better way to do this
	updatedScript := strings.Replace(string(scriptBytes), scriptLanguage(script).Header(), "", 1)

	projectFile := project.Get().Source()
	for i, projectScript := range projectFile.Scripts {
		if projectScript.Name == EditArgs.Name {
			projectFile.Scripts[i].Value = updatedScript
			break
		}
	}

	// TODO: Ensure we can save with comments
	// Currently saving the file when a script has a language value
	// will overwrite the language to our enum (example python3 -> 6)
	return projectFile.Save()

}
