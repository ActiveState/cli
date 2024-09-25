package checkoutinfo

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

type configurer interface {
	GetBool(string) bool
}

type projecter interface {
	Owner() string
	Name() string
	BranchName() string
	LegacyCommitID() string
	SetLegacyCommit(string) error
	Dir() string
	URL() string
}

type CheckoutInfo struct {
	auth    *authentication.Auth
	cfg     configurer
	project projecter
	svcm    *model.SvcModel
}

var ErrBuildscriptNotExist = errors.New("Build script does not exist")

type ErrInvalidCommitID struct {
	CommitID string
}

func (e ErrInvalidCommitID) Error() string {
	return "invalid commit ID"
}

func New(auth *authentication.Auth, cfg configurer, project projecter, svcm *model.SvcModel) *CheckoutInfo {
	return &CheckoutInfo{auth, cfg, project, svcm}
}

func (c *CheckoutInfo) Owner() string {
	return c.project.Owner()
}

func (c *CheckoutInfo) Name() string {
	return c.project.Name()
}

func (c *CheckoutInfo) Branch() string {
	return c.project.BranchName()
}

func (c *CheckoutInfo) CommitID() (strfmt.UUID, error) {
	if c.cfg.GetBool(constants.OptinBuildscriptsConfig) {
		if script, err := c.BuildScript(); err == nil {
			commitID, err2 := script.CommitID()
			if err2 != nil {
				return "", errs.Wrap(err, "Could not get commit ID from build script")
			}
			return commitID, nil
		} else {
			return "", errs.Wrap(err, "Could not get build script")
		}
	}

	// Read from activestate.yaml.
	commitID := c.project.LegacyCommitID()
	if !strfmt.IsUUID(commitID) {
		return "", &ErrInvalidCommitID{commitID}
	}

	return strfmt.UUID(commitID), nil
}

// CommitIDForReset will return either the commit ID from the buildscript, or the commitID from
// activestate.yaml, whichever one is valid.
// This should only be called by `state reset` for the purposes of resetting the commitID.
func (c *CheckoutInfo) CommitIDForReset() (strfmt.UUID, error) {
	if c.cfg.GetBool(constants.OptinBuildscriptsConfig) {
		if script, err := c.BuildScript(); err == nil {
			if commitID, err2 := script.CommitID(); err2 == nil {
				return commitID, nil
			}
		}
	}

	// Read from activestate.yaml.
	commitID := c.project.LegacyCommitID()
	if !strfmt.IsUUID(commitID) {
		return "", &ErrInvalidCommitID{commitID}
	}

	return strfmt.UUID(commitID), nil
}

func (c *CheckoutInfo) BuildScript() (*buildscript.BuildScript, error) {
	if !c.cfg.GetBool(constants.OptinBuildscriptsConfig) {
		bp := buildplanner.NewBuildPlannerModel(c.auth, c.svcm)
		script, err := bp.GetBuildScript(c.Owner(), c.Name(), c.Branch(), c.project.LegacyCommitID())
		if err != nil {
			return nil, errs.Wrap(err, "Could not get remote build script")
		}
		return script, nil
	}

	path := filepath.Join(c.project.Dir(), constants.BuildScriptFileName)

	data, err := fileutils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errs.Pack(err, ErrBuildscriptNotExist)
		}
		return nil, errs.Wrap(err, "Could not read build script from file")
	}

	script, err := buildscript.Unmarshal(data)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build script")
	}

	return script, nil
}

func (c *CheckoutInfo) SetCommitID(commitID strfmt.UUID) error {
	// Update commitID in activestate.yaml.
	logging.Debug("Updating commitID in activestate.yaml")
	if err := c.project.SetLegacyCommit(commitID.String()); err != nil {
		return errs.Wrap(err, "Could not set commit ID")
	}

	if !c.cfg.GetBool(constants.OptinBuildscriptsConfig) {
		return nil // buildscripts are not enabled, so nothing more to do
	}

	// Update commitID in Project field of build script.
	logging.Debug("Updating commitID in buildscript")
	buildscriptPath := filepath.Join(c.project.Dir(), constants.BuildScriptFileName)

	if !fileutils.FileExists(buildscriptPath) {
		return c.InitializeBuildScript(commitID)
	}

	data, err := fileutils.ReadFile(buildscriptPath)
	if err != nil {
		return errs.Wrap(err, "Could not read build script for updating")
	}

	script, err := buildscript.Unmarshal(data)
	if err != nil {
		if errors.Is(err, buildscript.ErrOutdatedAtTime) {
			return nil // likely running `state reset LOCAL`, so ignore this error
		}
		return errs.Wrap(err, "Could not unmarshal build script")
	}

	script.SetProjectURL(c.project.URL())

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

func (c *CheckoutInfo) InitializeBuildScript(commitID strfmt.UUID) error {
	if c.cfg.GetBool(constants.OptinBuildscriptsConfig) {
		buildplanner := buildplanner.NewBuildPlannerModel(c.auth, c.svcm)
		script, err := buildplanner.GetBuildScript(c.Owner(), c.Name(), c.Branch(), commitID.String())
		if err != nil {
			return errs.Wrap(err, "Unable to get the remote build script")
		}

		scriptBytes, err := script.Marshal()
		if err != nil {
			return errs.Wrap(err, "Unable to marshal build script")
		}

		scriptPath := filepath.Join(c.project.Dir(), constants.BuildScriptFileName)
		logging.Debug("Initializing build script at %s", scriptPath)
		err = fileutils.WriteFile(scriptPath, scriptBytes)
		if err != nil {
			return errs.Wrap(err, "Unable to write build script")
		}
	}

	// Update activestate.yaml.
	if err := c.project.SetLegacyCommit(commitID.String()); err != nil {
		return errs.Wrap(err, "Could not set commit ID")
	}

	return nil
}

func (c *CheckoutInfo) UpdateBuildScript(newScript *buildscript.BuildScript) error {
	if c.cfg.GetBool(constants.OptinBuildscriptsConfig) {
		script, err := c.BuildScript()
		if err != nil {
			return errs.Wrap(err, "Could not get local build script")
		}

		equals, err := script.Equals(newScript)
		if err != nil {
			return errs.Wrap(err, "Could not compare build script")
		}
		if script != nil && equals {
			return nil // no changes to write
		}

		sb, err := newScript.Marshal()
		if err != nil {
			return errs.Wrap(err, "Could not marshal build script")
		}

		logging.Debug("Writing build script")
		path := filepath.Join(c.project.Dir(), constants.BuildScriptFileName)
		if err := fileutils.WriteFile(filepath.Join(path, constants.BuildScriptFileName), sb); err != nil {
			return errs.Wrap(err, "Could not write build script to file")
		}
	}

	// Update activestate.yaml.
	commitID, err := newScript.CommitID()
	if err != nil {
		return errs.Wrap(err, "Could not get script commit ID")
	}
	if err := c.project.SetLegacyCommit(commitID.String()); err != nil {
		return errs.Wrap(err, "Could not set commit ID")
	}

	return nil
}
