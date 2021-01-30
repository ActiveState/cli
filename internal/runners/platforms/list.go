package platforms

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// ListRunParams tracks the info required for running List.
type ListRunParams struct {
	Project *project.Project
}

// List manages the listing execution context.
type List struct {
	out output.Outputer
}

// NewList prepares a list execution context for use.
func NewList(prime primer.Outputer) *List {
	return &List{
		out: prime.Output(),
	}
}

// Run executes the list behavior.
func (l *List) Run(ps ListRunParams) error {
	logging.Debug("Execute platforms list")

	listing, err := newListing("", ps.Project.Name(), ps.Project.Owner(), ps.Project.BranchName())
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

func newListing(commitID, projName, projOrg, projBranch string) (*Listing, error) {
	targetCommitID, err := targetedCommitID(commitID, projName, projOrg, projBranch)
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

func targetedCommitID(commitID, projName, projOrg, projBranch string) (*strfmt.UUID, error) {
	if commitID != "" {
		var cid strfmt.UUID
		err := cid.UnmarshalText([]byte(commitID))

		return &cid, err
	}

	latest, err := model.LatestCommitID(projOrg, projName, projBranch)
	if err != nil {
		return nil, err
	}

	return latest, nil
}
