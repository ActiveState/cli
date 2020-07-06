package show

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
	prj "github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Params describes the data required for the show run func.
type Params struct {
	Remote string
}

// Show manages the show run execution context.
type Show struct {
	project *project.Project
	out     output.Outputer
}

// New returns a pointer to an instance of Show.
func New(pj *project.Project, out output.Outputer) *Show {
	return &Show{
		project: pj,
		out:     out,
	}
}

// Run is the primary show logic.
func (s *Show) Run(params Params) error {
	logging.Debug("Execute")

	if s.project == nil {
		return locale.NewError("err_no_projectfile")
	}

	pj := s.project

	if params.Remote != "" {
		path := params.Remote
		projectFilePath := filepath.Join(params.Remote, constants.ConfigFileName)

		if _, err := os.Stat(path); err != nil {
			return locale.WrapError(
				err,
				"err_state_show_path_does_not_exist",
				"Directory does not exist.",
			)
		}

		if _, err := os.Stat(projectFilePath); err != nil {
			return locale.WrapError(
				err,
				"err_state_show_no_config",
				"activestate.yaml file not found in the given location.",
			)
		}

		projectFile, err := projectfile.Parse(projectFilePath)
		if err != nil {
			logging.Errorf("Unable to parse activestate.yaml: %s", err)
			return locale.WrapError(
				err,
				"err_state_show_project_parse",
				"Could not parse activestate.yaml.",
			)
		}

		var fail *failures.Failure
		pj, fail = prj.New(projectFile)
		if fail != nil {
			return fail.ToError()
		}
	}

	src := pj.Source()

	updater.PrintUpdateMessage(src.Path())

	data := outputData{
		Namespace:    pj.Namespace(),
		Name:         pj.Name(),
		Organization: pj.Owner(),
		Platforms:    platformsData(src),
		Languages:    languagesData(src),
		Events:       eventsData(src),
		Scripts:      scriptsData(src),
	}

	s.out.Print(data)

	return nil
}

type outputData struct {
	Namespace    string
	Name         string
	Organization string
	Platforms    []string          `json:",omitempty"`
	Languages    []string          `json:",omitempty"`
	Events       []string          `json:",omitempty"`
	Scripts      map[string]string `json:",omitempty"`
}

func platformsData(project *projectfile.Project) []string {
	if len(project.Platforms) == 0 {
		return nil
	}

	var data []string
	for _, platform := range project.Platforms {
		constrained := "*"
		if !constraints.PlatformMatches(platform) {
			constrained = ""
		}
		v := fmt.Sprintf("%s%s %s %s (%s)", constrained, platform.Os, platform.Version, platform.Architecture, platform.Name)
		data = append(data, v)
	}

	return data
}

func eventsData(project *projectfile.Project) []string {
	if len(project.Events) == 0 {
		return nil
	}

	es := projectfile.MakeEventsFromConstrainedEntities(
		constraints.FilterUnconstrained(project.Events.AsConstrainedEntities()),
	)

	var data []string
	for _, event := range es {
		data = append(data, event.Name)
	}

	return data
}

func scriptsData(project *projectfile.Project) map[string]string {
	if len(project.Scripts) == 0 {
		return nil
	}

	scripts := projectfile.MakeScriptsFromConstrainedEntities(
		constraints.FilterUnconstrained(project.Scripts.AsConstrainedEntities()),
	)

	data := make(map[string]string)
	for _, script := range scripts {
		data[script.Name] = script.Description
	}

	return data
}

func languagesData(project *projectfile.Project) []string {
	if len(project.Languages) == 0 {
		return nil
	}

	languages := projectfile.MakeLanguagesFromConstrainedEntities(
		constraints.FilterUnconstrained(project.Languages.AsConstrainedEntities()),
	)

	var data []string
	for _, language := range languages {
		data = append(data, fmt.Sprintf("%s %s", language.Name, language.Version))
	}

	return data
}
