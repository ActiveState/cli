package buildscript

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

func getRemoteBuildScript(proj *project.Project, customCommit strfmt.UUID, auth *authentication.Auth) (*bpModel.BuildScript, error) {
	bp := model.NewBuildPlanner(auth)
	commitID := proj.CommitUUID()
	if customCommit != "" {
		commitID = customCommit
	}
	return bp.GetBuildScript(proj.Owner(), proj.Name(), commitID.String())
}

func NeedsUpdate(proj *project.Project, customCommit strfmt.UUID, auth *authentication.Auth) (bool, error) {
	of, err := buildscript.Get(proj.Dir())
	if err != nil {
		if buildscript.IsDoesNotExistError(err) {
			return false, nil
		}
		return false, errs.Wrap(err, "Could not get local build script")
	}

	script, err := getRemoteBuildScript(proj, customCommit, auth)
	if err != nil {
		return false, errs.Wrap(err, "Could not get remote build script")
	}

	return of.Script.Equals(script), nil
}

func Update(proj *project.Project, customCommit strfmt.UUID, out output.Outputer, auth *authentication.Auth) (strfmt.UUID, error) {
	of, err := buildscript.Get(proj.Dir())
	if err != nil {
		return "", errs.Wrap(err, "Could not get local build script")
	}
	script, err := getRemoteBuildScript(proj, customCommit, auth)
	if err != nil {
		return "", errs.Wrap(err, "Could not get remote build script")
	}

	out.Notice(locale.Tl("buildscript_update", "Updating project to reflect build script changes..."))

	bp := model.NewBuildPlanner(auth)
	commit, err := bp.PushCommit(model.PushCommitParams{
		Owner:        proj.Owner(),
		Project:      proj.Name(),
		ParentCommit: proj.CommitID(),
		BranchRef:    proj.BranchName(),
		Description:  locale.Tl("buildscript_commit_description", "Update project due to build script change."),
		Script:       of.Script,
	})
	if err != nil {
		return "", errs.Wrap(err, "Could not update project to reflect build script changes.")
	}
	commitID := strfmt.UUID(commit.CommitID)

	// commitID will be in its own file after DX-1852, so proj will no longer be changed.
	if err := proj.SetCommit(commitID.String()); err != nil {
		return "", errs.Wrap(err, "Could not update project file commit ID.")
	}

	script, err = getRemoteBuildScript(proj, commitID, auth)
	if err != nil {
		return "", errs.Wrap(err, "Could not get remote build script")
	}

	if err := of.Update(script); err != nil {
		return "", errs.Wrap(err, "Could not update local build script.")
	}

	return commitID, nil
}

func UpdateIfNeeded(proj *project.Project, out output.Outputer, auth *authentication.Auth) error {
	needsUpdate, err := NeedsUpdate(proj, "", auth)
	if err != nil {
		return errs.Wrap(err, "Could not check if local build script needs updating")
	}
	if !needsUpdate {
		return nil
	}
	_, err2 := Update(proj, "", out, auth)
	if err2 != nil {
		return errs.Wrap(err, "Could not update build script")
	}
	return nil
}
