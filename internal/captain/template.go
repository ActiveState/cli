package captain

import (
	"fmt"
	"strings"
	"text/template"
	"unicode"

	"github.com/ActiveState/cli/internal/locale"
)

func localizedTemplate(c *Command, template string) string {
	localizedArgs := []map[string]string{}
	for _, arg := range c.Arguments() {
		req := ""
		if arg.Required {
			req = "1"
		}
		localizedArgs = append(localizedArgs, map[string]string{
			"Name":        arg.Name,
			"Description": arg.Description,
			"Required":    req,
		})
	}
	return locale.Tt(template, map[string]interface{}{
		"Arguments": localizedArgs,
		"isRootCmd": c.Use() == "state",
	})
}

func templateFuncs(cmd *Command) template.FuncMap {
	return template.FuncMap{
		"trimTrailingWhitespaces": trimTrailingWhitespaces,
		"rpad":                    rpad,
		"commandGroups":           func() []CommandGroup { return cmd.groups },
		"findCmd":                 func(name string) *Command { return cmd.FindSafe([]string{name}) },
	}
}

func findCmd(c *Command, name string) *Command {
	return c.FindSafe([]string{name})
}

func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

func trimTrailingWhitespaces(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}
