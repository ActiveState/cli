package localcommit

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// proj holds the project instance most recently accessed, if any.
// Using globals in this way is an anti-pattern, but because the commit mechanic is going through a lot of changes
// we're currently handling it this way to help further refactors. Once we've landed the go-forward mechanic we should
// remove this anti-pattern.
// https://activestatef.atlassian.net/browse/DX-2524
var proj *project.Project

type ErrInvalidCommitID struct {
	CommitID string
}

func (e ErrInvalidCommitID) Error() string {
	return "invalid commit ID"
}

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

	commitID := proj.LegacyCommitID()
	if !strfmt.IsUUID(commitID) {
		return "", &ErrInvalidCommitID{commitID}
	}

	return strfmt.UUID(commitID), nil
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

	// Instead of passing a config around, test for buildscript presence. If it exists, assume
	// buildscripts are enabled and update its Project field.
	if fileutils.FileExists(filepath.Join(proj.Dir(), constants.BuildScriptFileName)) {
		if err := updateBuildScript(proj); err != nil {
			return errs.Wrap(err, "Could not update build script")
		}
	}

	return nil
}

func updateBuildScript(pj *project.Project) error {
	scriptPath := filepath.Join(pj.Dir(), constants.BuildScriptFileName)

	data, err := fileutils.ReadFile(scriptPath)
	if err != nil {
		return errs.Wrap(err, "Could not read build script from file")
	}

	script, err := buildscript.Unmarshal(data)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshal build script")
	}

	if pj.URL() == script.Project() {
		return nil //nothing to update
	}
	script.SetProject(pj.URL())

	data, err = script.Marshal()
	if err != nil {
		return errs.Wrap(err, "Could not marshal build script")
	}

	if err := fileutils.WriteFile(scriptPath, data); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}

	return nil
}
