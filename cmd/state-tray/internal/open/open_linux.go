package open

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

// Prompt brings up the user's preferred shell within a new terminal.
func Prompt(command string) error {
	shell, err := preferredShell()
	if err != nil {
		logging.Errorf("Preferred shell failure (falling back to bash): %v", err)
		shell = "bash"
	}

	command = fmt.Sprintf("%s;%s", command, shell)
	cmd := exec.Command("x-terminal-emulator", "-e", shell, "-c", command)
	if err := cmd.Run(); err != nil {
		return locale.WrapError(err, "err_open_prompt", "Could not open prompt")
	}

	return nil
}

func preferredShell() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", locale.WrapError(err, "err_user_unknown", "Cannot get current user")
	}

	// searching /etc/passwd is the standard way to find one's default shell
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return "", locale.WrapError(err, "err_open_passwd", "Cannot open user info file")
	}
	defer f.Close()

	prefix := currentUser.Name + ":"
	var shell string

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		parts := strings.Split(line, ":")
		shell = parts[len(parts)-1]
	}
	if err := sc.Err(); err != nil {
		return "", locale.WrapError(err, "err_scan_passwd", "Error parsing user info file")
	}

	if shell == "" {
		return "", locale.NewError("err_shell_unknown", "No preferred shell obtained")
	}

	if !fileutils.IsExecutable(shell) || fileutils.IsDir(shell) {
		return "", locale.NewError("err_shell_not_exec", "Preferred shell cannot execute")
	}

	return shell, nil
}
