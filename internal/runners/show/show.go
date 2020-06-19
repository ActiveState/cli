package show

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bndr/gotabulate"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/updater"
	prj "github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// RunParams describes the data required for the show run func.
type RunParams struct {
	Remote string
}

// Show manages the show run execution context.
type Show struct {
	out output.Outputer
}

// New returns a pointer to an instance of Show.
func New(out output.Outputer) *Show {
	return &Show{
		out: out,
	}
}

// Run is the primary show logic.
func (s *Show) Run(params RunParams) error {
	logging.Debug("Execute")

	var project *prj.Project
	if params.Remote == "" {
		project = prj.Get()
	} else {
		path := params.Remote
		projectFilePath := filepath.Join(params.Remote, constants.ConfigFileName)
		if _, err := os.Stat(path); err != nil {
			print.Error(locale.T("err_state_show_path_does_not_exist"))
			return nil
		} else if _, err := os.Stat(projectFilePath); err != nil {
			print.Error(locale.T("err_state_show_no_config"))
			return nil
		}
		projectFile, err := projectfile.Parse(projectFilePath)
		if err != nil {
			logging.Errorf("Unable to parse activestate.yaml: %s", err)
			print.Error(locale.T("err_state_show_project_parse"))
			return nil
		}
		var fail *failures.Failure
		project, fail = prj.New(projectFile)
		if fail != nil {
			failures.Handle(fail.ToError(), fail.Message)
			return nil
		}
	}

	updater.PrintUpdateMessage(project.Source().Path())

	outType := s.out.Type()
	switch outType {
	case output.JSONFormatName, output.EditorV0FormatName, output.EditorFormatName:
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

	return nil
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
