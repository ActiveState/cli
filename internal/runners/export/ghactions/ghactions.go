package ghactions

import (
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
)

type GithubActions struct {
	project *project.Project
	output  output.Outputer
}

type Params struct{}

type primeable interface {
	primer.Projecter
	primer.Outputer
}

func New(primer primeable) *GithubActions {
	return &GithubActions{primer.Project(), primer.Output()}
}

func (g *GithubActions) Run(p *Params) error {
	jobs := g.project.Jobs()
	if len(jobs) == 0 {
		return locale.NewInputError("err_ghac_nojobs", "In order to export a github actions workflow you must first create at least one 'job' in your activestate.yaml.")
	}

	workflow := Workflow{
		Name: locale.Tl("ghac_workflow_name", "State Tool Generated Workflow"),
		On: WorkflowTrigger{
			Push:        WorkflowBranches{Branches: []string{"master"}},
			PullRequest: WorkflowBranches{Branches: []string{"master"}},
		},
		Jobs: map[string]WorkflowJob{},
	}
	for _, job := range jobs {
		workflowJob := WorkflowJob{
			RunsOn: "ubuntu-latest",
			Env:    map[string]interface{}{},
			Steps: []WorkflowStep{
				{
					Name: "Checkout",
					Uses: "actions/checkout@v2",
				},
				{
					Name: "Install State Tool",
					Run:  `sh <(curl -q https://platform.activestate.com/dl/cli/install.sh) -n -f`,
				},
			},
		}
		for _, constant := range job.Constants() {
			v, err := constant.Value()
			if err != nil {
				return locale.WrapError(err, "err_ghac_constant", "Could not get value for constant: {{.V0}}.", constant.Name())
			}
			workflowJob.Env[constant.Name()] = v
		}
		for _, script := range job.Scripts() {
			workflowJob.Steps = append(workflowJob.Steps, WorkflowStep{
				Name: script.Name(),
				Run:  "state run " + script.Name(),
			})
		}
		workflow.Jobs[job.Name()] = workflowJob
	}

	out, err := yaml.Marshal(workflow)
	if err != nil {
		return locale.NewError("err_ghac_marshal", "Failed to create yaml file: {{.V0}}", err.Error())
	}

	g.output.Print(string(out))
	return nil
}
