//+build linux

package open

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

// Prompt brings up the user's preferred shell within a new terminal.
func Prompt(command string) error {
	shell, err := preferredShellWithFallback("/bin/bash")
	if err != nil {
		return locale.WrapError(err, "err_get_shell", "Cannot get preferred shell")
	}

	command = fmt.Sprintf("%s;%s", command, shell)
	cmd := exec.Command("x-terminal-emulator", "-e", shell, "-c", command)
	if err := cmd.Run(); err != nil {
		return locale.WrapError(err, "err_open_prompt", "Could not open prompt")
	}

	return nil
}

func preferredShellWithFallback(fallback string) (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", locale.WrapError(err, "err_user_unknown", "Cannot get current user")
	}

	f, err := os.Open("/etc/passwd")
	if err != nil {
		return "", locale.WrapError(err, "err_open_passwd", "Cannot open passwd file")
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
		return "", locale.WrapError(err, "err_scan_passwd", "/etc/passwd file scan failed")
	}

	if shell == "" {
		shell = fallback
	}

	return shell, nil
}
