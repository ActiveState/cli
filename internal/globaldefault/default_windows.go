package globaldefault

import (
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/subshell/cmd"
)

func isOnPATH(binDir string) bool {
	cmdEnv := cmd.NewCmdEnv(true)
	path, err := cmdEnv.Get("PATH")
	if err != nil {
		multilog.Error("Failed to get user PATH")
		return false
	}

	return true /*funk.ContainsString(
		strings.Split(path, string(os.PathListSeparator)),
		binDir,
	)*/
}
