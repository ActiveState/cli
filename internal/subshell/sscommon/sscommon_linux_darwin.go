package sscommon

import (
	"strings"

	"github.com/ActiveState/cli/internal/osutils"
)

func RunFuncByBinary(binary string) RunFunc {
	bin := strings.ToLower(binary)
	if strings.Contains(bin, "bash") {
		return runWithBash
	}
	return runDirect
}

func runWithBash(env []string, name string, args ...string) (int, error) {
	filePath, fail := osutils.BashifyPath(name)
	if fail != nil {
		return 1, fail.ToError()
	}

	esc := osutils.NewBashEscaper()

	quotedArgs := filePath
	for _, arg := range args {
		quotedArgs += " " + esc.Quote(arg)
	}

	return runDirect(env, "bash", "-c", quotedArgs)
}