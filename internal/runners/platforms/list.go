package platforms

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// List manages the listing execution context.
type List struct {
	out  output.Outputer
	proj *project.Project
}

// NewList prepares a list execution context for use.
func NewList(prime primeable) *List {
	return &List{
		out:  prime.Output(),
		proj: prime.Project(),
	}
}

// Run executes the list behavior.
func (l *List) Run() error {
	logging.Debug("Execute platforms list")

	if l.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	listing, err := newListing("", l.proj.Name(), l.proj.Owner(), l.proj.BranchName())
	if err != nil {
		return err
	}

	l.out.Print(listing)
	return nil
}

// Listing represents the output data of a listing.
type Listing struct {
	Platforms []*Platform `json:"platforms"`
}

func newListing(commitID, projName, projOrg string, branchName string) (*Listing, error) {
	targetCommitID, err := targetedCommitID(commitID, projName, projOrg, branchName)
	if err != nil {
		return nil, err
	}

	platforms, err := model.FetchPlatformsForCommit(*targetCommitID)
	if err != nil {
		return nil, err
	}

	listing := Listing{
		Platforms: makePlatformsFromModelPlatforms(platforms),
	}

	return &listing, nil
}

func targetedCommitID(commitID, projName, projOrg, branchName string) (*strfmt.UUID, error) {
	if commitID != "" {
		var cid strfmt.UUID
		err := cid.UnmarshalText([]byte(commitID))

		return &cid, err
	}

	latest, err := model.BranchCommitID(projOrg, projName, branchName)
	if err != nil {
		return nil, err
	}

	return latest, nil
}
