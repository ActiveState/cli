package globaldefault

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/thoas/go-funk"
)

func isOnPATH(binDir string) bool {
	cmdEnv := cmd.NewCmdEnv(true)
	path, err := cmdEnv.Get("PATH")
	if err != nil {
		logging.Error("Failed to get user PATH")
		return false
	}

	return funk.ContainsString(
		strings.Split(path, string(os.PathListSeparator)),
		binDir,
	)
}
