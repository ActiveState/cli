package captain

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"unicode"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/spf13/cobra"
)

type Templater struct {
	Cmd           *Command
	CommandGroups []CommandGroup
}

func (t *Templater) RootCmdUsageFunc() func(c *cobra.Command) error {
	return func(c *cobra.Command) error {
		tpl := template.New("root_usage_tpl")
		tpl.Funcs(t.templateFuncs())
		template.Must(tpl.New("usage_tpl").Parse(localizedTemplate(t.Cmd, "usage_tpl")))
		template.Must(tpl.Parse(locale.Tt("root_usage_tpl")))
		return tpl.Execute(os.Stdout, c)
	}
}

func (t *Templater) defaultUsageFunc() func(c *cobra.Command) error {
	return func(c *cobra.Command) error {
		tpl := template.New("usage_tpl")
		tpl.Funcs(t.templateFuncs())
		template.Must(tpl.Parse(localizedTemplate(t.Cmd, "usage_tpl")))
		return tpl.Execute(os.Stdout, c)
	}
}

func (t *Templater) cmdGroupsString(c *cobra.Command) string {
	var groups []string
	for _, cmdGroup := range t.cmdGroups(c) {
		cmds := []string{cmdGroup.Message}
		for _, cmd := range cmdGroup.Commands {
			if cmd.cobra.IsAvailableCommand() {
				cmds = append(cmds, "  "+rpad(cmd.Use(), cmd.cobra.NamePadding())+" "+cmd.Description())
			}
		}
		groups = append(groups, strings.Join(cmds, "\n"))
	}
	return strings.Join(groups, "\n\n")
}

func (t *Templater) cmdGroups(c *cobra.Command) []CommandGroup {
	if len(t.CommandGroups) == 0 {
		return nil
	}
	return t.CommandGroups
}

func (t *Templater) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"CmdGroupsString":         t.cmdGroupsString,
		"trimTrailingWhitespaces": trimTrailingWhitespaces,
		"rpad":                    rpad,
	}
}

func localizedTemplate(c *Command, template string) string {
	localizedArgs := []map[string]string{}
	for _, arg := range c.Arguments() {
		req := ""
		if arg.Required {
			req = "1"
		}
		localizedArgs = append(localizedArgs, map[string]string{
			"Name":        locale.T(arg.Name),
			"Description": locale.T(arg.Description),
			"Required":    req,
		})
	}
	return locale.Tt(template, map[string]interface{}{
		"Arguments": localizedArgs,
		"isRootCmd": c.Use() == "state",
	})
}

func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

func trimTrailingWhitespaces(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}
