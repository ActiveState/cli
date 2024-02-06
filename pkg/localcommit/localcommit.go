package localcommit

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// proj holds the project instance most recently accessed, if any.
// Using globals in this way is an anti-pattern, but because the commit mechanic is going through a lot of changes
// we're currently handling it this way to help further refactors. Once we've landed the go-forward mechanic we should
// remove this anti-pattern.
var proj *project.Project

func setupProject(pjpath string) error {
	if proj != nil && proj.Dir() == pjpath {
		return nil
	}
	var err error
	proj, err = project.FromPath(pjpath)
	if err != nil {
		return errs.Wrap(err, "Could not get project info to set up project")
	}
	return nil
}

func Get(pjpath string) (strfmt.UUID, error) {
	if err := setupProject(pjpath); err != nil {
		return "", errs.Wrap(err, "Could not setup project")
	}

	if !strfmt.IsUUID(proj.LegacyCommitID()) {
		return "", locale.NewInputError("err_commit_id_invalid", "Commit ID in activestate.yaml is not UUID formatted: {{.V0}}.", proj.LegacyCommitID())
	}

	return strfmt.UUID(proj.LegacyCommitID()), nil
}

func Set(pjpath, commitID string) error {
	if !strfmt.IsUUID(commitID) {
		return locale.NewInputError("err_commit_id_invalid", commitID)
	}

	if err := setupProject(pjpath); err != nil {
		return errs.Wrap(err, "Could not setup project")
	}

	if err := proj.SetLegacyCommit(commitID); err != nil {
		return errs.Wrap(err, "Could not set commit ID")
	}

	return nil
}
