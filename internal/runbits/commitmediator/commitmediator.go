package commitmediator

import (
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/runbits/legacy/projectmigration"
	"github.com/ActiveState/cli/pkg/localcommit"
)

type projecter interface {
	Source() *projectfile.Project
	Dir() string
	URL() string
	Path() string
	LegacyCommitID() string
	StripLegacyCommitID() error
	SetLegacyCommit(string) error
}

// Get returns the given project's commit ID in either the new format (commit file), or the old
// format (activestate.yaml).
// If you require the commit file, use localcommit.Get().
func Get(proj projecter) (strfmt.UUID, error) {
	if commitID, err := localcommit.Get(proj.Dir()); err == nil {
		if proj.LegacyCommitID() != "" {
			if err := projectmigration.Warn(proj); err != nil {
				return "", errs.Wrap(err, "Could not warn about migration")
			}
		}
		return commitID, nil
	} else if localcommit.IsFileDoesNotExistError(err) {
		if err := projectmigration.PromptAndMigrate(proj); err != nil {
			return "", errs.Wrap(err, "Could not prompt and/or migrate project")
		}
		return localcommit.Get(proj.Dir())
	} else {
		return "", errs.Wrap(err, "Could not get local commit")
	}
}

func Set(proj projecter, commitID string) error {
	if proj.LegacyCommitID() != "" {
		if err := proj.SetLegacyCommit(commitID); err != nil {
			return errs.Wrap(err, "Could not set legacy commit")
		}
	}
	return localcommit.Set(proj.Dir(), commitID)
}
