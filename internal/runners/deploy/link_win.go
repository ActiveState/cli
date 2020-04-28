// +build windows

package deploy

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func link(src, dst string) error {
	if strings.HasSuffix(dst, ".exe") {
		dst = strings.Replace(dst, ".exe", ".lnk", 1)
	}
	logging.Debug("Creating shortcut, oldname: %s newname: %s", src, dst)

	root, err := environment.GetRootPath()
	if err != nil {
		return locale.WrapError(
			err, "err_link_get_root",
			"Could not get root path of shortcut script",
		)
	}
	scriptPath := filepath.Join(root, "assets", "scripts", "createShortcut.ps1")

	// Some paths may contain spaces so we must quote
	src = strconv.Quote(src)
	dst = strconv.Quote(dst)

	cmd := exec.Command("powershell.exe", "-Command", scriptPath, src, dst)
	return cmd.Run()
}
