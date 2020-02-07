package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// List manages the listing execution context.
type List struct{}

// NewList prepares a list execution context for use.
func NewList() *List {
	return &List{}
}

// Run executes the list behavior.
func (l *List) Run() (*Listing, error) {
	logging.Debug("Execute platforms list")

	return newListing("")
}

// Listing represents the output data of a listing.
type Listing struct {
	Platforms []*Platform `json:"platforms"`
}

func newListing(commitID string) (*Listing, error) {
	targetCommitID, err := targettedCommitID(commitID)
	if err != nil {
		return nil, err
	}

	platforms, fail := model.FetchPlatformsForCommit(targetCommitID)
	if fail != nil {
		return nil, fail
	}

	listing := Listing{
		Platforms: makePlatformsFromModelPlatforms(platforms),
	}

	return &listing, nil
}

func targettedCommitID(commitID string) (strfmt.UUID, error) {
	if commitID == "" {
		proj := project.Get()
		cmt, fail := model.LatestCommitID(proj.Owner(), proj.Name())
		if fail != nil {
			return strfmt.UUID(""), fail
		}
		commitID = cmt.String()
	}

	var cid strfmt.UUID
	err := cid.UnmarshalText([]byte(commitID))

	return cid, err
}
