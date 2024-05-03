package osutils

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/skratchdot/open-golang/open"
)

// The default open command and editors based on platform
const (
	openCmdLin       = "xdg-open"
	openCmdMac       = "open"
	defaultEditorWin = "notepad.exe"
)

func OpenEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return open.Start(filename)
	}

	err := verifyEditor(editor)
	if err != nil {
		return errs.Wrap(err, "Failed to verify editor: %s", editor)
	}

	return openEditor(filename, editor)
}

func openEditor(filename, editorCmd string) error {
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

func verifyEditor(editor string) error {
	if strings.Contains(editor, string(os.PathSeparator)) {
		return verifyPathEditor(editor)
	}

	_, err := exec.LookPath(editor)
	if err != nil {
		return errs.Wrap(err, "Failed to find a suite-able editor.")
	}

	return nil
}

func verifyPathEditor(editor string) error {
	if runtime.GOOS == "windows" && filepath.Ext(editor) == "" {
		return locale.NewInputError(
			"error_edit_windows_invalid_editor",
			"Editor path must contain a file extension")
	}

	_, err := os.Stat(editor)
	if err != nil {
		return locale.WrapInputError(err, "error_edit_stat_editor", "Failed to find editor '{{.V0}}' on file system.", editor)
	}

	return nil
}
