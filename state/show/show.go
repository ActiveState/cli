package show

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/scm"
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

	print.Formatted("Name: %s\n", project.Name)
	print.Formatted("Organization: %s\n", project.Owner)
	print.Formatted("URL: %s\n", "")
	print.Formatted("Platforms:\n")
	for _, platform := range project.Platforms {
		print.Formatted("  %s %s %s (%s)\n", platform.Os, platform.Version, platform.Architecture, platform.Name)
	}
	print.Formatted("Hooks:\n")
	for _, hook := range project.Hooks {
		print.Formatted("  %s: %s\n", hook.Name, hook.Value)
	}
	print.Formatted("Commands:\n")
	for _, command := range project.Commands {
		print.Formatted("  %s: %s\n", command.Name, command.Value)
	}
	print.Formatted("Languages:\n")
	for _, language := range project.Languages {
		print.Formatted("  %s %s (%d packages)\n", language.Name, language.Version, len(language.Packages))
	}
	print.Formatted("Environment variables:\n")
	for _, variable := range project.Variables {
		print.Formatted("  %s = %s\n", variable.Name, variable.Value)
	}
}
