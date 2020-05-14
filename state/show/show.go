package show

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	prj "github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
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

// Flags holds global flags passed through the command line.
var Flags struct {
	Output *string
}

// Execute the show command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	var project *prj.Project
	if Args.Remote == "" {
		project = prj.Get()
	} else {
		path := Args.Remote
		projectFilePath := filepath.Join(Args.Remote, constants.ConfigFileName)
		if _, err := os.Stat(path); err != nil {
			print.Error(locale.T("err_state_show_path_does_not_exist"))
			return
		} else if _, err := os.Stat(projectFilePath); err != nil {
			print.Error(locale.T("err_state_show_no_config"))
			return
		}
		projectFile, err := projectfile.Parse(projectFilePath)
		if err != nil {
			logging.Errorf("Unable to parse activestate.yaml: %s", err)
			print.Error(locale.T("err_state_show_project_parse"))
			return
		}
		var fail *failures.Failure
		project, fail = prj.New(projectFile)
		if fail != nil {
			failures.Handle(fail.ToError(), fail.Message)
			return
		}
	}

	updater.PrintUpdateMessage(project.Source().Path())

	output := commands.Output(strings.ToLower(*Flags.Output))
	switch output {
	case commands.JSON, commands.EditorV0:
		print.Line(fmt.Sprintf("{\"namespace\": \"%s/%s\"}", project.Owner(), project.Name()))
	default:
		print.BoldInline("%s: ", locale.T("print_state_show_name"))
		print.Line("%s", project.Name())

		print.BoldInline("%s: ", locale.T("print_state_show_organization"))
		print.Line("%s", project.Owner())

		print.Line("")

		printPlatforms(project.Source())
		printLanguages(project.Source())
		printScripts(project.Source())
		printEvents(project.Source())
	}
}

func printPlatforms(project *projectfile.Project) {
	if len(project.Platforms) == 0 {
		return
	}

	rows := [][]interface{}{}
	for _, platform := range project.Platforms {
		constrained := "*"
		if !constraints.PlatformMatches(platform) {
			constrained = ""
		}
		v := fmt.Sprintf("%s%s %s %s (%s)", constrained, platform.Os, platform.Version, platform.Architecture, platform.Name)
		rows = append(rows, []interface{}{v})
	}

	print.BoldInline("%s:", locale.T("print_state_show_platforms"))
	printTable(rows)
}

func printEvents(project *projectfile.Project) {
	if len(project.Events) == 0 {
		return
	}

	rows := [][]interface{}{}
	es := projectfile.MakeEventsFromConstrainedEntities(
		constraints.FilterUnconstrained(project.Events.AsConstrainedEntities()),
	)
	for _, event := range es {
		rows = append(rows, []interface{}{event.Name})
	}

	print.BoldInline("%s:", locale.T("print_state_show_events"))
	printTable(rows)
}

func printScripts(project *projectfile.Project) {
	if len(project.Scripts) == 0 {
		return
	}

	rows := [][]interface{}{}
	scripts := projectfile.MakeScriptsFromConstrainedEntities(
		constraints.FilterUnconstrained(project.Scripts.AsConstrainedEntities()),
	)
	for _, script := range scripts {
		rows = append(rows, []interface{}{script.Name, script.Description})
	}

	print.BoldInline("%s:", locale.T("print_state_show_scripts"))
	printTable(rows)
}

func printLanguages(project *projectfile.Project) {
	if len(project.Languages) == 0 {
		return
	}

	rows := [][]interface{}{}
	languages := projectfile.MakeLanguagesFromConstrainedEntities(
		constraints.FilterUnconstrained(project.Languages.AsConstrainedEntities()),
	)
	for _, language := range languages {
		rows = append(rows, []interface{}{language.Name, language.Version})
	}

	print.BoldInline("%s:", locale.T("print_state_show_languages"))
	printTable(rows)
}

func printTable(rows [][]interface{}) {
	t := gotabulate.Create(rows)

	// gotabulate tries to make the first row the headers, so use some empty header instead
	// this is also the reason why we're using BoldInLine, since the header line will act as the newline
	t.SetHeaders([]string{""})

	t.SetHideLines([]string{"betweenLine", "top", "aboveTitle", "belowheader", "LineTop", "LineBottom", "bottomLine"}) // Don't print whitespace lines
	t.SetAlign("left")
	print.Line(t.Render("plain"))
}
