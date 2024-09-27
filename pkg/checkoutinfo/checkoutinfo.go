package checkoutinfo

import (
	"github.com/go-openapi/strfmt"
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
	SetLegacyCommit(string) error
}

type CheckoutInfo struct {
	project projectfiler
}

func New(project projectfiler) *CheckoutInfo {
	return &CheckoutInfo{project}
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

func (c *CheckoutInfo) SetCommitID(commitID strfmt.UUID) error {
	return c.project.SetLegacyCommit(commitID.String())
}
