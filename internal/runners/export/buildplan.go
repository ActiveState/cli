package export

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
)

type BuildPlanParams struct {
	Namespace *project.Namespaced
	CommitID  string
	Target    string
}

type BuildPlan struct {
	prime primeable
}

func NewBuildPlan(p primeable) *BuildPlan {
	return &BuildPlan{p}
}

func (b *BuildPlan) Run(params *BuildPlanParams) (rerr error) {
	defer rationalizeError(&rerr, b.prime.Auth())

	proj := b.prime.Project()
	out := b.prime.Output()
	if proj != nil && !params.Namespace.IsValid() {
		out.Notice(locale.Tr("operating_message", proj.NamespaceString(), proj.Dir()))
	}

	commit, err := buildplanner.GetCommit(
		params.Namespace, params.CommitID, params.Target, b.prime)
	if err != nil {
		return errs.Wrap(err, "Could not get commit")
	}

	bytes, err := commit.BuildPlan().Marshal()
	if err != nil {
		return errs.Wrap(err, "Could not marshal build plan")
	}
	expr := make(map[string]interface{})
	err = json.Unmarshal(bytes, &expr)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshal build plan")
	}

	out.Print(output.Prepare(string(bytes), expr))

	return nil
}
