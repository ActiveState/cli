package services

import (
	"golang.org/x/net/context"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/idl"
	"github.com/ActiveState/cli/internal/logging"
)

type Version struct {
	idl.UnsafeVersionSvcServer // We use Unsafe because we actually want our code to complain if endpoints aren't implemented
}

func NewVersion() *Version {
	return &Version{}
}

func (v *Version) StateVersion(ctx context.Context, in *idl.StateVersionRequest) (*idl.StateVersionResponse, error) {
	logging.Debug("Received SayHello")
	return &idl.StateVersionResponse{
		License:  constants.LibraryLicense,
		Version:  constants.Version,
		Branch:   constants.BranchName,
		Revision: constants.RevisionHash,
		Date:     constants.Date,
	}, nil
}
