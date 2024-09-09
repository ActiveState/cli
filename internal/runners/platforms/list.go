package platforms

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/checkoutinfo"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// List manages the listing execution context.
type List struct {
	out  output.Outputer
	proj *project.Project
	auth *authentication.Auth
}

// NewList prepares a list execution context for use.
func NewList(prime primeable) *List {
	return &List{
		out:  prime.Output(),
		proj: prime.Project(),
		auth: prime.Auth(),
	}
}

// Run executes the list behavior.
func (l *List) Run() error {
	logging.Debug("Execute platforms list")

	if l.proj == nil {
		return rationalize.ErrNoProject
	}

	commitID, err := checkoutinfo.GetCommitID(l.proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}

	targetCommitID, err := targetedCommitID(commitID.String(), l.proj.Name(), l.proj.Owner(), l.proj.BranchName())
	if err != nil {
		return errs.Wrap(err, "Unable to get commit ID")
	}

	modelPlatforms, err := model.FetchPlatformsForCommit(*targetCommitID, l.auth)
	if err != nil {
		return errs.Wrap(err, "Unable to get platforms for commit")
	}

	platforms := makePlatformsFromModelPlatforms(modelPlatforms)
	var plainOutput interface{} = platforms
	if len(platforms) == 0 {
		plainOutput = locale.Tl("platforms_list_no_platforms", "There are no platforms for this project.")
	}
	l.out.Print(output.Prepare(plainOutput, platforms))
	return nil
}

func targetedCommitID(commitID, projName, projOrg, branchName string) (*strfmt.UUID, error) {
	if commitID != "" {
		var cid strfmt.UUID
		err := cid.UnmarshalText([]byte(commitID))

		return &cid, err
	}

	latest, err := model.BranchCommitID(projOrg, projName, branchName)
	if err != nil {
		return nil, err
	}

	return latest, nil
}
