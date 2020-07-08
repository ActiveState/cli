package show

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
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
	project     *project.Project
	out         output.Outputer
	conditional *constraints.Conditional
}

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Conditioner
}

// New returns a pointer to an instance of Show.
func New(prime primeable) *Show {
	return &Show{
		prime.Project(),
		prime.Output(),
		prime.Conditional(),
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

	events, err := eventsData(src, s.conditional)
	if err != nil {
		return locale.WrapError(err, "err_show_events", "Could not parse events.")
	}

	scripts, err := scriptsData(src, s.conditional)
	if err != nil {
		return locale.WrapError(err, "err_show_scripts", "Could not parse scripts.")
	}

	data := outputData{
		Namespace:    pj.Namespace(),
		Name:         pj.Name(),
		Organization: pj.Owner(),
		Events:       events,
		Scripts:      scripts,
	}

	s.out.Print(data)

	return nil
}

type outputData struct {
	Namespace    string
	Name         string
	Organization string
	Events       []string          `json:",omitempty"`
	Scripts      map[string]string `json:",omitempty"`
}

func eventsData(project *projectfile.Project, conditional *constraints.Conditional) ([]string, error) {
	if len(project.Events) == 0 {
		return nil, nil
	}

	constrained, err := constraints.FilterUnconstrained(conditional, project.Events.AsConstrainedEntities())
	if err != nil {
		return nil, locale.WrapError(err, "err_event_condition", "Event has invalid conditional")
	}

	es := projectfile.MakeEventsFromConstrainedEntities(constrained)

	var data []string
	for _, event := range es {
		data = append(data, event.Name)
	}

	return data, nil
}

func scriptsData(project *projectfile.Project, conditional *constraints.Conditional) (map[string]string, error) {
	if len(project.Scripts) == 0 {
		return nil, nil
	}

	constrained, err := constraints.FilterUnconstrained(conditional, project.Scripts.AsConstrainedEntities())
	if err != nil {
		return nil, locale.WrapError(err, "err_script_condition", "Script has invalid conditional")
	}

	scripts := projectfile.MakeScriptsFromConstrainedEntities(constrained)

	data := make(map[string]string)
	for _, script := range scripts {
		data[script.Name] = script.Description
	}

	return data, nil
}
