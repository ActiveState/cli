package checkoutinfo

import (
	"errors"
	"os"
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

func GetCommitID(pjpath string) (strfmt.UUID, error) {
	if err := setupProject(pjpath); err != nil {
		return "", errs.Wrap(err, "Could not setup project")
	}

	commitID := proj.LegacyCommitID()
	if !strfmt.IsUUID(commitID) {
		return "", &ErrInvalidCommitID{commitID}
	}

	return strfmt.UUID(commitID), nil
}

func SetCommitID(pjpath, commitID string) error {
	if !strfmt.IsUUID(commitID) {
		return locale.NewInputError("err_commit_id_invalid", commitID)
	}

	if err := setupProject(pjpath); err != nil {
		return errs.Wrap(err, "Could not setup project")
	}

	if err := proj.SetLegacyCommit(commitID); err != nil {
		return errs.Wrap(err, "Could not set commit ID")
	}

	if err := updateBuildScript(); err != nil {
		return errs.Wrap(err, "Could not update build script")
	}

	return nil
}

// updateBuildScript updates the build script's Project info field.
// Note: cannot use runbits.buildscript.ScriptFromProject() and Update() due to import cycle.
func updateBuildScript() error {
	buildscriptPath := filepath.Join(proj.ProjectDir(), constants.BuildScriptFileName)

	data, err := fileutils.ReadFile(buildscriptPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// There is no build script to update, so just exit.
			// Normally we would put this behind a optin.buildscripts config test, but that would require
			// another config global anti-pattern for this package.
			return nil // no build script to update
		}
		return errs.Wrap(err, "Could not read build script for updating")
	}

	script, err := buildscript.Unmarshal(data)
	if err != nil {
		if errors.Is(err, buildscript.ErrOutdatedAtTime) {
			return nil // likely running `state reset LOCAL`, so ignore this error
		}
		return errs.Wrap(err, "Could not unmarshal build script")
	}

	script.SetProjectURL(proj.URL())

	data, err = script.Marshal()
	if err != nil {
		return errs.Wrap(err, "Could not marshal updated build script")
	}

	err = fileutils.WriteFile(buildscriptPath, data)
	if err != nil {
		return errs.Wrap(err, "Could not write updated build script")
	}

	return nil
}

func UpdateProject(script *buildscript.BuildScript, dir string) error {
	err := setupProject(dir)
	if err != nil {
		return errs.Wrap(err, "Could not setup project")
	}

	proj.Source().Project = script.ProjectURL()
	err = proj.Source().Save(nil)
	if err != nil {
		return errs.Wrap(err, "Could not update project")
	}

	return nil
}
