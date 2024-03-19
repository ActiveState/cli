package buildscript

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func getBuildExpression(proj *project.Project, customCommit *strfmt.UUID, auth *authentication.Auth) (*buildexpression.BuildExpression, error) {
	bp := model.NewBuildPlannerModel(auth)
	commitID, err := localcommit.Get(proj.Dir())
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get local commit ID")
	}
	if customCommit != nil {
		commitID = *customCommit
	}
	return bp.GetBuildExpression(commitID.String())
}

// Sync synchronizes the local build script with the remote one.
// If a commit ID is given, a local mutation has occurred (e.g. added a package, pulled, etc.), so
// pull in the new build script. Otherwise, if there are local build script changes, create a new
// commit with them in order to update the remote one.
func Sync(proj *project.Project, commitID *strfmt.UUID, out output.Outputer, auth *authentication.Auth) (synced bool, err error) {
	logging.Debug("Synchronizing local build script using commit %s", commitID)
	script, err := buildscript.ScriptFromProjectWithFallback(proj, auth)
	if err != nil {
		return false, errs.Wrap(err, "Could not get local build script")
	}

	expr, err := getBuildExpression(proj, commitID, auth)
	if err != nil {
		return false, errs.Wrap(err, "Could not get remote build expr for provided commit")
	}

	// If commitID is given, a mutation happened, so prefer the remote build expression.
	// Otherwise, prefer local changes.
	if script != nil && commitID == nil {
		logging.Debug("Checking for changes")
		if script.EqualsBuildExpression(expr) {
			return false, nil // nothing to do
		}
		logging.Debug("Merging changes")
		out.Notice(locale.Tl("buildscript_update", "Updating project to reflect build script changes..."))

		localCommitID, err := localcommit.Get(proj.Dir())
		if err != nil {
			return false, errs.Wrap(err, "Unable to get local commit ID")
		}

		expr, err := script.BuildExpression()
		if err != nil {
			return false, errs.Wrap(err, "Unable to get build expression from build script")
		}

		bp := model.NewBuildPlannerModel(auth)
		stagedCommitID, err := bp.StageCommit(model.StageCommitParams{
			Owner:        proj.Owner(),
			Project:      proj.Name(),
			ParentCommit: localCommitID.String(),
			Expression:   expr,
		})
		if err != nil {
			return false, errs.Wrap(err, "Could not update project to reflect build script changes.")
		}
		commitID = &stagedCommitID

		localcommit.Set(proj.Dir(), commitID.String())

		_, err = getBuildExpression(proj, commitID, auth) // timestamps might be different
		if err != nil {
			return false, errs.Wrap(err, "Could not get remote build expr for staged commit")
		}

		synced = true
	}

	if err := buildscript.Update(proj, expr, auth); err != nil {
		return false, errs.Wrap(err, "Could not update local build script.")
	}

	return synced, nil
}

func generateDiff(script *buildscript.Script, expr *buildexpression.BuildExpression) (string, error) {
	newScript, err := buildscript.NewScriptFromBuildExpression(expr)
	if err != nil {
		return "", errs.Wrap(err, "Unable to transform build expression to build script")
	}

	local := locale.Tl("diff_local", "local")
	remote := locale.Tl("diff_remote", "remote")

	var result bytes.Buffer

	diff := diffmatchpatch.New()
	scriptLines, newScriptLines, lines := diff.DiffLinesToChars(script.String(), newScript.String())
	hunks := diff.DiffMain(scriptLines, newScriptLines, false)
	hunks = diff.DiffCharsToLines(hunks, lines)
	hunks = diff.DiffCleanupSemantic(hunks)
	for i := 0; i < len(hunks); i++ {
		switch hunk := hunks[i]; hunk.Type {
		case diffmatchpatch.DiffEqual:
			result.WriteString(hunk.Text)
		case diffmatchpatch.DiffDelete:
			result.WriteString(fmt.Sprintf("<<<<<<< %s\n", local))
			result.WriteString(hunk.Text)
			result.WriteString("=======\n")
			if i+1 < len(hunks) && hunks[i+1].Type == diffmatchpatch.DiffInsert {
				result.WriteString(hunks[i+1].Text)
				i++ // do not process this hunk again
			}
			result.WriteString(fmt.Sprintf(">>>>>>> %s\n", remote))
		case diffmatchpatch.DiffInsert:
			result.WriteString(fmt.Sprintf("<<<<<<< %s\n", local))
			result.WriteString("=======\n")
			result.WriteString(hunk.Text)
			result.WriteString(fmt.Sprintf(">>>>>>> %s\n", remote))
		}
	}

	return result.String(), nil
}

func GenerateAndWriteDiff(proj *project.Project, script *buildscript.Script, expr *buildexpression.BuildExpression) error {
	result, err := generateDiff(script, expr)
	if err != nil {
		return errs.Wrap(err, "Could not generate diff between local and remote build scripts")
	}
	return fileutils.WriteFile(filepath.Join(proj.Dir(), constants.BuildScriptFileName), []byte(result))
}
