package commitmediator

import (
	"github.com/go-openapi/strfmt"
)

type projecter interface {
	Dir() string
	URL() string
	Path() string
	LegacyCommitID() string
	LegacySetCommit(string) error // remove this in DX-2307
}

// Get returns the given project's commit ID in either the new format (commit file), or the old
// format (activestate.yaml).
// If you require the commit file, use localcommit.Get().
func Get(proj projecter) (strfmt.UUID, error) {
	// Re-enable the contents of this function in DX-2307
	//if commitID, err := localcommit.Get(proj.Dir()); err == nil {
	//	return commitID, nil
	//} else if localcommit.IsFileDoesNotExistError(err) {
	//if migrated, err := projectmigration.PromptAndMigrate(proj); err == nil && migrated {
	//	return localcommit.Get(proj.Dir())
	//} else if err != nil {
	//	return "", errs.Wrap(err, "Could not prompt and/or migrate project")
	//}
	return strfmt.UUID(proj.LegacyCommitID()), nil
	//} else {
	//	return "", errs.Wrap(err, "Could not get local commit")
	//}
}

func Set(proj projecter, commitID string) error {
	// Replace all calls to this function with localcommit.Set() in DX-2307.
	// Also, consider changing localcommit.Set() to accept a projecter interface with Dir().
	return proj.LegacySetCommit(commitID)
}
