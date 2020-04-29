package platforms

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

var (
	// FailNoCommitID indicates that no commit id is provided and not
	// obtainable from the current project.
	FailNoCommitID = failures.Type("platforms.fail.nocommitid", failures.FailNonFatal)
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
func NewList(out output.Outputer) *List {
	return &List{
		out: out,
	}
}

// Run executes the list behavior.
func (l *List) Run(ps ListRunParams) error {
	logging.Debug("Execute platforms list")

	listing, err := newListing("", ps.Project.Name(), ps.Project.Owner())
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

func newListing(commitID, projName, projOrg string) (*Listing, error) {
	targetCommitID, err := targetedCommitID(commitID, projName, projOrg)
	if err != nil {
		return nil, err
	}

	platforms, fail := model.FetchPlatformsForCommit(*targetCommitID)
	if fail != nil {
		return nil, fail
	}

	listing := Listing{
		Platforms: makePlatformsFromModelPlatforms(platforms),
	}

	return &listing, nil
}

func targetedCommitID(commitID, projName, projOrg string) (*strfmt.UUID, error) {
	if commitID != "" {
		var cid strfmt.UUID
		err := cid.UnmarshalText([]byte(commitID))

		return &cid, err
	}

	latest, fail := model.LatestCommitID(projOrg, projName)
	if fail != nil {
		return nil, fail.ToError()
	}

	return latest, nil
}
