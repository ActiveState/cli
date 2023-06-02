package buildscript

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

func getBuildExpression(proj *project.Project, customCommit *strfmt.UUID, auth *authentication.Auth) (*bpModel.BuildExpression, error) {
	bp := model.NewBuildPlanner(auth)
	commitID := proj.CommitUUID()
	if customCommit != nil {
		commitID = *customCommit
	}
	return bp.GetBuildExpression(proj.Owner(), proj.Name(), commitID.String())
}

// Sync synchronizes the local build script with the remote one.
// If a commit ID is given, a local mutation has occurred (e.g. added a package, pulled, etc.), so
// pull in the new build script. Otherwise, if there are local build script changes, create a new
// commit with them in order to update the remote one.
func Sync(proj *project.Project, commitID *strfmt.UUID, out output.Outputer, auth *authentication.Auth) error {
	logging.Debug("Synchronizing local build script")
	script, err := buildscript.NewScriptFromProjectDir(proj.Dir())
	if err != nil && !buildscript.IsDoesNotExistError(err) {
		return errs.Wrap(err, "Could not get local build script")
	}

	expr, err := getBuildExpression(proj, commitID, auth)
	if err != nil {
		return errs.Wrap(err, "Could not get remote build expr")
	}

	if script != nil {
		logging.Debug("Checking for changes")
		if script.Equals(expr) {
			return nil // nothing to do
		}
		logging.Debug("Merging changes")
		// Note: merging and/or conflict resolution will happen in another ticket.
		// For now, if commitID is given, a mutation happened, so prefer the remote build expression.
		// Otherwise, prefer local changes.
		if commitID == nil {
			expr, err = bpModel.NewBuildExpression([]byte(script.String()))
			if err != nil {
				return errs.Wrap(err, "Unable to translate local build script to build expression")
			}
		}

		out.Notice(locale.Tl("buildscript_update", "Updating project to reflect build script changes..."))

		bp := model.NewBuildPlanner(auth)
		stagedCommitID, err := bp.StageCommit(model.StateCommitParams{
			Owner:        proj.Owner(),
			Project:      proj.Name(),
			ParentCommit: proj.CommitID(),
			Script:       expr,
		})
		if err != nil {
			return errs.Wrap(err, "Could not update project to reflect build script changes.")
		}
		commitID = &stagedCommitID

		expr, err = getBuildExpression(proj, commitID, auth) // timestamps might be different
		if err != nil {
			return errs.Wrap(err, "Could not get remote build expr")
		}
	}

	if err := buildscript.UpdateOrCreate(proj.Dir(), expr); err != nil {
		return errs.Wrap(err, "Could not update local build script.")
	}

	// For target.ProjectTargets that have already been set up without a custom commit ID, update the
	// project's commit ID so that runtime setup uses the correct commit ID.
	// Note: this should no longer be required once DX-1852 lands (commit ID will be in its own file).
	if commitID != nil {
		if err := proj.SetCommit(commitID.String()); err != nil {
			return errs.Wrap(err, "Could not update project file commit ID.")
		}
	}

	return nil
}
