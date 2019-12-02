// +build !windows

package activate

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/virtualenvironment"
)

func output() error {
	venv := virtualenvironment.Get()
	fail := venv.Activate()
	if fail != nil {
		return fail
	}

	env := virtualenvironment.Get().GetEnvSlice(true)
	envJSON := make([]string, len(env))
	for i, kv := range env {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			continue
		}
		envJSON[i] = fmt.Sprintf("\"%s\": \"%s\"", kv[:eq], kv[eq+1:])
	}

	fmt.Printf("{ %s }\n", strings.Join(envJSON, ", "))
	return nil
}