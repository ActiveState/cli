// +build !windows

package sscommon

import (
	"strings"

	"github.com/ActiveState/cli/internal/osutils"
)

var escaper *osutils.ShellEscape

func init() {
	escaper = osutils.NewBashEscaper()
}

// EscapeEnv escapes all values so they can be exported
func EscapeEnv(env map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range env {
		result[k] = v
		result[k] = escaper.Escape(result[k])
		result[k] = strings.ReplaceAll(result[k], lineBreak, lineBreakChar)
	}
	return result
}
