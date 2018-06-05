package variables

import (
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var calls int // for preventing infinite recursion during recursively expansion

// ExpandFromProject searches for $category.name-style variables in the given
// string and substitutes them with their contents, derived from the given
// project, and subject to the given constraints (if any).
func ExpandFromProject(s string, p *projectfile.Project) string {
	calls++
	if calls > 10 {
		calls = 0 // reset
		return ""
	}
	regex := regexp.MustCompile("\\${?\\w+\\.\\w+}?")
	expanded := regex.ReplaceAllStringFunc(s, func(variable string) string {
		components := strings.Split(strings.TrimLeft(variable, "$"), ".")
		category := components[0]
		name := components[1]
		var value string
		switch category {
		case "platform":
			for _, platform := range p.Platforms {
				if !constraints.PlatformMatches(platform) {
					continue
				}
				switch name {
				case "name":
					value = platform.Name
				case "os":
					value = platform.Os
				case "version":
					value = platform.Version
				case "architecture":
					value = platform.Architecture
				case "libc":
					value = platform.Libc
				case "compiler":
					value = platform.Compiler
				}
			}
		case "variables":
			for _, variable := range p.Variables {
				if variable.Name == name && !constraints.IsConstrained(variable.Constraints) {
					value = variable.Value
					break
				}
			}
		case "hooks":
			for _, hook := range p.Hooks {
				if hook.Name == name && !constraints.IsConstrained(hook.Constraints) {
					value = hook.Value
					break
				}
			}
		case "commands":
			for _, command := range p.Commands {
				if command.Name == name && !constraints.IsConstrained(command.Constraints) {
					value = command.Value
					break
				}
			}
		}
		return ExpandFromProject(value, p)
	})
	calls--
	return expanded
}
