package checkoutinfo

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/buildscript"
)

type ErrInvalidCommitID struct {
	CommitID string
}

func (e ErrInvalidCommitID) Error() string {
	return "invalid commit ID"
}

type projectfiler interface {
	Owner() string
	Name() string
	BranchName() string
	LegacyCommitID() string
	SetNamespace(string, string) error
	SetBranch(string) error
	SetLegacyCommit(string) error
	Dir() string
	URL() string
}

type CheckoutInfo struct {
	project           projectfiler
	optinBuildScripts bool
}

var ErrBuildscriptNotExist = errors.New("Build script does not exist")

func New(project projectfiler, optinBuildScripts bool) *CheckoutInfo {
	return &CheckoutInfo{project, optinBuildScripts}
}

// Owner returns the project owner from activestate.yaml.
// Note: cannot read this from buildscript because it may not exist yet.
func (c *CheckoutInfo) Owner() string {
	return c.project.Owner()
}

// Name returns the project name from activestate.yaml.
// Note: cannot read this from buildscript because it may not exist yet.
func (c *CheckoutInfo) Name() string {
	return c.project.Name()
}

// Branch returns the project branch from activestate.yaml.
// Note: cannot read this from buildscript because it may not exist yet.
func (c *CheckoutInfo) Branch() string {
	return c.project.BranchName()
}

func (c *CheckoutInfo) CommitID() (strfmt.UUID, error) {
	commitID := c.project.LegacyCommitID()
	if !strfmt.IsUUID(commitID) {
		return "", &ErrInvalidCommitID{commitID}
	}
	return strfmt.UUID(commitID), nil
}

func (c *CheckoutInfo) SetNamespace(owner, project string) error {
	err := c.project.SetNamespace(owner, project)
	if err != nil {
		return errs.Wrap(err, "Unable to update project")
	}
	return c.updateBuildScriptProject()
}

func (c *CheckoutInfo) SetBranch(branch string) error {
	err := c.project.SetBranch(branch)
	if err != nil {
		return errs.Wrap(err, "Unable to update project")
	}
	return c.updateBuildScriptProject()
}

func (c *CheckoutInfo) SetCommitID(commitID strfmt.UUID) error {
	err := c.project.SetLegacyCommit(commitID.String())
	if err != nil {
		return errs.Wrap(err, "Unable to update project")
	}
	return c.updateBuildScriptProject()
}

func (c *CheckoutInfo) updateBuildScriptProject() error {
	if !c.optinBuildScripts {
		return nil
	}

	scriptPath := filepath.Join(c.project.Dir(), constants.BuildScriptFileName)
	data, err := fileutils.ReadFile(scriptPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errs.Pack(err, ErrBuildscriptNotExist)
		}
		return errs.Wrap(err, "Could not read build script from file")
	}

	script, err := buildscript.Unmarshal(data)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshal build script")
	}

	if c.project.URL() == script.Project() {
		return nil //nothing to update
	}
	script.SetProject(c.project.URL())

	data, err = script.Marshal()
	if err != nil {
		return errs.Wrap(err, "Could not marshal build script")
	}

	if err := fileutils.WriteFile(scriptPath, data); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}

	return nil
}
