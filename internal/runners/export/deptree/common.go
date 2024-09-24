package deptree

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

func resolveNamespace(inputNs *project.Namespaced, inputCommitID string, prime primeable) (*project.Namespaced, error) {
	out := prime.Output()
	proj := prime.Project()
	if proj == nil {
		return nil, rationalize.ErrNoProject
	}

	ns := inputNs
	dir := "https://" + constants.PlatformURL + "/" + ns.String()
	if !ns.IsValid() {
		ns = proj.Namespace()
		dir = proj.Dir()
	}

	commitID := strfmt.UUID(inputCommitID)
	if commitID == "" {
		if inputNs.IsValid() {
			p, err := model.FetchProjectByName(ns.Owner, ns.Project, prime.Auth())
			if err != nil {
				return nil, errs.Wrap(err, "Unable to get project")
			}
			branch, err := model.DefaultBranchForProject(p)
			if err != nil {
				return nil, errs.Wrap(err, "Could not grab branch for project")
			}
			if branch.CommitID == nil {
				return nil, errs.New("branch has not commit")
			}
			ns.CommitID = branch.CommitID
		} else {
			var err error
			commitID, err = localcommit.Get(proj.Dir())
			if err != nil {
				return nil, errs.Wrap(err, "Unable to get local commit ID")
			}
			ns.CommitID = &commitID
		}
	}

	out.Notice(locale.Tr("operating_message", ns.String(), dir))

	return ns, nil
}
