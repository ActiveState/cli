package buildscript

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression/merge"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// Merge merges the local build script with the remote buildexpression (not script) for a given
// UUID, performing the given merge strategy (e.g. from model.MergeCommit).
func Merge(proj *project.Project, remoteCommit strfmt.UUID, strategies *mono_models.MergeStrategies, auth *authentication.Auth) error {
	// Verify we have a build script to merge.
	script, err := buildscript.NewScriptFromProjectDir(proj.Dir())
	if err != nil && !buildscript.IsDoesNotExistError(err) {
		return errs.Wrap(err, "Could not get local build script")
	}
	if script == nil {
		return nil // no build script to merge
	}

	// Get the local and remote build expressions to merge.
	bp := model.NewBuildPlannerModel(auth)
	localCommit, err := localcommit.Get(proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	exprA, err := bp.GetBuildExpression(proj.Owner(), proj.Name(), localCommit.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get buildexpression for local commit")
	}
	exprB, err := bp.GetBuildExpression(proj.Owner(), proj.Name(), remoteCommit.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get buildexpression for remote commit")
	}

	// Attempt the merge.
	mergedExpr, err := merge.Merge(exprA, exprB, strategies)
	if err != nil {
		if errs.Matches(err, &merge.AutoMergeNotPossibleError{}) || errs.Matches(err, &merge.MergeConflictsError{}) {
			err := GenerateAndWriteDiff(proj, script, exprB)
			if err != nil {
				return locale.WrapError(err, "err_diff_build_script", "Unable to generate differences between local and remote build script")
			}
			return locale.NewInputError("err_build_script_merge", "Unable to automatically merge build scripts")
		}
		return errs.Wrap(err, "Unable to merge buildexpressions")
	}

	// Write the merged build expression as a local build script.
	return buildscript.UpdateOrCreate(proj.Dir(), mergedExpr)
}
