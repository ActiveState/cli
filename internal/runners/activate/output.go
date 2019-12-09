package activate

import (
	"fmt"
	"strings"
)

func formatJSON(env []string) string {
	cleaned := make([]string, len(env))
	for i, e := range env {
		cleaned[i] = strings.ReplaceAll(e, "\\", "\\\\")
	}
	return fmt.Sprintf("{ %s }", strings.Join(cleaned, ", "))
}
