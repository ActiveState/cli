package buildplanner

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type ErrInvalidCommitId struct {
	Id string
}

func (e *ErrInvalidCommitId) Error() string {
	return "Invalid commit ID"
}

type ErrCommitDoesNotExistInProject struct {
	Project  string
	CommitID string
}

func (e *ErrCommitDoesNotExistInProject) Error() string {
	return "Commit does not exist in project"
}

type primeable interface {
	primer.Projecter
	primer.Auther
	primer.Outputer
	primer.SvcModeler
}

// GetCommit returns a commit from the given arguments. By default, the local commit for the
// current project is returned, but a commit for a given commitID for the current project can be
// returned, as can the commit for a remote project (and optional commitID).
func GetCommit(
	namespace *project.Namespaced,
	commitID string,
	target string,
	prime primeable,
) (commit *bpModel.Commit, rerr error) {
	pj := prime.Project()
	out := prime.Output()
	auth := prime.Auth()
	svcm := prime.SvcModel()

	if pj == nil && !namespace.IsValid() {
		return nil, rationalize.ErrNoProject
	}

	commitUUID := strfmt.UUID(commitID)
	if commitUUID != "" && !strfmt.IsUUID(commitUUID.String()) {
		return nil, &ErrInvalidCommitId{commitUUID.String()}
	}

	namespaceProvided := namespace.IsValid()
	commitIdProvided := commitUUID != ""

	// Show a spinner when fetching a buildplan.
	// Sourcing the local runtime for a buildplan has its own spinner.
	pb := output.StartSpinner(out, locale.T("progress_solve"), constants.TerminalAnimationInterval)
	defer func() {
		message := locale.T("progress_success")
		if rerr != nil {
			message = locale.T("progress_fail")
		}
		pb.Stop(message + "\n") // extra empty line
	}()

	targetPtr := ptr.To(request.TargetAll)
	if target != "" {
		targetPtr = &target
	}

	var err error
	switch {
	// Return the buildplan from this runtime.
	case !namespaceProvided && !commitIdProvided:
		localCommitID, err := localcommit.Get(pj.Path())
		if err != nil {
			return nil, errs.Wrap(err, "Could not get local commit")
		}

		bp := bpModel.NewBuildPlannerModel(auth, svcm)
		commit, err = bp.FetchCommit(localCommitID, pj.Owner(), pj.Name(), targetPtr)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch commit")
		}

	// Return buildplan from the given commitID for the current project.
	case !namespaceProvided && commitIdProvided:
		bp := bpModel.NewBuildPlannerModel(auth, svcm)
		commit, err = bp.FetchCommit(commitUUID, pj.Owner(), pj.Name(), targetPtr)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch commit")
		}

	// Return the buildplan for the latest commitID of the given project.
	case namespaceProvided && !commitIdProvided:
		pj, err := model.FetchProjectByName(namespace.Owner, namespace.Project, auth)
		if err != nil {
			return nil, locale.WrapExternalError(err, "err_fetch_project", "", namespace.String())
		}

		branch, err := model.DefaultBranchForProject(pj)
		if err != nil {
			return nil, errs.Wrap(err, "Could not grab branch for project")
		}

		branchCommitUUID, err := model.BranchCommitID(namespace.Owner, namespace.Project, branch.Label)
		if err != nil {
			return nil, errs.Wrap(err, "Could not get commit ID for project")
		}
		commitUUID = *branchCommitUUID

		bp := bpModel.NewBuildPlannerModel(auth, svcm)
		commit, err = bp.FetchCommit(commitUUID, namespace.Owner, namespace.Project, targetPtr)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch commit")
		}

	// Return the buildplan for the given commitID of the given project.
	case namespaceProvided && commitIdProvided:
		bp := bpModel.NewBuildPlannerModel(auth, svcm)
		commit, err = bp.FetchCommit(commitUUID, namespace.Owner, namespace.Project, targetPtr)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch commit")
		}

	default:
		return nil, errs.New("Unhandled case")
	}

	// Note: the Platform does not raise an error when requesting a commit ID that does not exist in
	// a given project, so we have verify existence client-side. See DS-1705 (yes, DS, not DX).
	var owner, name, nsString string
	var localCommitID *strfmt.UUID
	if namespaceProvided {
		owner = namespace.Owner
		name = namespace.Project
		nsString = namespace.String()
	} else {
		owner = pj.Owner()
		name = pj.Name()
		nsString = pj.NamespaceString()
		commitID, err := localcommit.Get(pj.Path())
		if err != nil {
			return nil, errs.Wrap(err, "Could not get local commit")
		}
		localCommitID = &commitID
	}
	_, err = model.GetCommitWithinProjectHistory(commit.CommitID, owner, name, localCommitID, auth)
	if err != nil {
		if err == model.ErrCommitNotInHistory {
			return nil, errs.Pack(err, &ErrCommitDoesNotExistInProject{nsString, commit.CommitID.String()})
		}
		return nil, errs.Wrap(err, "Unable to determine if commit exists in project")
	}

	return commit, nil
}

// GetBuildPlan returns a project's buildplan, depending on the given arguments. By default, the
// buildplan for the current project is returned, but a buildplan for a given commitID for the
// current project can be returned, as can the buildplan for a remote project (and optional
// commitID).
func GetBuildPlan(
	namespace *project.Namespaced,
	commitID string,
	target string,
	prime primeable,
) (bp *buildplan.BuildPlan, rerr error) {
	commit, err := GetCommit(namespace, commitID, target, prime)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get commit")
	}
	return commit.BuildPlan(), nil
}
