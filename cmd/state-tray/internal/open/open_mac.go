//+build darwin

package open

import (
	"os/exec"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/scriptfile"
)

func Prompt(command string) error {
	sf, err := scriptfile.New(language.Bash, "open-terminal", command)
	if err != nil {
		return locale.WrapError(err, "err_open_create_scriptfile", "Could not create temporary script file")
	}

	cmd := exec.Command("open", "-a", getPrompt(), sf.Filename())
	err = cmd.Run()
	if err != nil {
		return locale.WrapError(err, "err_open_prompt", "Could not open prompt")
	}

	return nil
}

func getPrompt() string {
	if fileutils.DirExists("/Applications/iTerm.app") {
		return "iTerm"
	}
	return "Terminal"
}
