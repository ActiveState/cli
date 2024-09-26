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
	LegacyCommitID() string
	SetLegacyCommit(string) error
}

type CheckoutInfo struct {
	project projectfiler
}

func New(project projectfiler) *CheckoutInfo {
	return &CheckoutInfo{project}
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
