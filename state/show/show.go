package show

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/scm"
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"
)

// Command is the show command's definition.
var Command = &commands.Command{
	Name:        "show",
	Description: "show_project",
	Run:         Execute,

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "remote",
			Description: "arg_state_show_remote_description",
			Variable:    &Args.Remote,
		},
	},
}

// Args holds the arg values passed through the command line.
var Args struct {
	Remote string
}

// Execute the show command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	var project *projectfile.Project
	if Args.Remote == "" {
		project = projectfile.Get()
	} else if scm := scm.FromRemote(Args.Remote); scm != nil {
		// TODO: remote fetching of activestate.yaml and parsing
	} else {
		path := Args.Remote
		projectFile := filepath.Join(Args.Remote, constants.ConfigFileName)
		if _, err := os.Stat(path); err != nil {
			print.Error(locale.T("err_state_show_path_does_not_exist"))
			return
		} else if _, err := os.Stat(projectFile); err != nil {
			print.Error(locale.T("err_state_show_no_config"))
			return
		}
		var err error
		project, err = projectfile.Parse(projectFile)
		if err != nil {
			logging.Errorf("Unable to parse activestate.yaml: %s", err)
			print.Error(locale.T("err_state_show_project_parse"))
			return
		}
	}

	print.BoldInline("%s: ", locale.T("print_state_show_name"))
	print.Formatted("%s\n", project.Name)

	print.BoldInline("%s: ", locale.T("print_state_show_organization"))
	print.Formatted("%s\n", project.Owner)

	//print.Bold("%s: \n", locale.T("print_state_show_url"))
	//print.Formatted("%s\n", "")

	if len(project.Platforms) > 0 {
		print.Bold("%s:", locale.T("print_state_show_platforms"))
		for _, platform := range project.Platforms {
			constrained := "*"
			if !constraints.PlatformMatches(platform) {
				constrained = " "
			}
			print.Formatted(" %s%s %s %s (%s)\n", constrained, platform.Os, platform.Version, platform.Architecture, platform.Name)
		}
	}

	if len(project.Hooks) > 0 {
		print.Bold("%s:", locale.T("print_state_show_hooks"))
		for _, hook := range project.Hooks {
			if !constraints.IsConstrained(hook.Constraints) {
				value, fail := variables.ExpandFromProject(hook.Value, project)
				if fail != nil {
					value = fail.Error()
				}
				print.Formatted("  %s: %s\n", hook.Name, value)
			}
		}
	}

	if len(project.Commands) > 0 {
		print.Bold("%s:", locale.T("print_state_show_commands"))
		for _, command := range project.Commands {
			if !constraints.IsConstrained(command.Constraints) {
				value, fail := variables.ExpandFromProject(command.Value, project)
				if fail != nil {
					value = fail.Error()
				}
				print.Formatted("  %s: %s\n", command.Name, value)
			}
		}
	}

	if len(project.Languages) > 0 {
		print.Bold("%s:", locale.T("print_state_show_languages"))
		for _, language := range project.Languages {
			if !constraints.IsConstrained(language.Constraints) {
				print.Formatted("  %s %s (%d %s)\n", language.Name, language.Version, len(language.Packages), locale.T("print_state_show_packages"))
			}
		}
	}

	if len(project.Variables) > 0 {
		print.Bold("%s:", locale.T("print_state_show_env_vars"))
		for _, variable := range project.Variables {
			if !constraints.IsConstrained(variable.Constraints) {
				value, fail := variables.ExpandFromProject(variable.Value, project)
				if fail != nil {
					value = fail.Error()
				}
				print.Formatted("  %s: %s\n", variable.Name, value)
			}
		}
	}
}
