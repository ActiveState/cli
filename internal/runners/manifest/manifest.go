package manifest

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	Output() output.Outputer
	Project() *project.Project
	Auth() *authentication.Auth
}

type Manifest struct {
	out     output.Outputer
	project *project.Project
	auth    *authentication.Auth
}

func NewManifest(prime primeable) *Manifest {
	return &Manifest{
		out:     prime.Output(),
		project: prime.Project(),
		auth:    prime.Auth(),
	}
}

func (m *Manifest) Run() (rerr error) {
	if m.project == nil {
		return rationalize.ErrNoProject
	}

	m.out.Notice(locale.Tl("manifest_operating_on_project", "Operating on project: [ACTIONABLE]{{.V0}}[/RESET], located at [ACTIONABLE]{{.V1}}[/RESET]\n", m.project.Namespace().String(), m.project.Dir()))

	commitID, err := localcommit.Get(m.project.Dir())
	if err != nil {
		return errs.Wrap(err, "Could not get commit ID")
	}

	bp := model.NewBuildPlannerModel(m.auth)
	expr, _, err := bp.GetBuildExpressionAndTime(commitID.String())
	if err != nil {
		return errs.Wrap(err, "Could not get remote build expr and time")
	}

	exprReqs, err := expr.Requirements()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements")
	}

	reqs, err := newRequirementsOutput(exprReqs, m.auth)
	if err != nil {
		return errs.Wrap(err, "Could not get requirements output")
	}

	m.out.Print(reqs)

	return nil
}
