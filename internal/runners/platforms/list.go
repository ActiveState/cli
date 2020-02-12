package platforms

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
)

var (
	// FailNoCommitID indicates that no commit id is provided and not
	// obtainable from the current project.
	FailNoCommitID = failures.Type("platforms.fail.nocommitid", failures.FailNonFatal)
)

// List manages the listing execution context.
type List struct {
	GetProject ProjectProviderFunc
}

// NewList prepares a list execution context for use.
func NewList(getProjFn ProjectProviderFunc) *List {
	return &List{
		GetProject: getProjFn,
	}
}

// Run executes the list behavior.
func (l *List) Run() (*Listing, error) {
	logging.Debug("Execute platforms list")

	return newListing("", l.GetProject)
}

// Listing represents the output data of a listing.
type Listing struct {
	Platforms []*Platform `json:"platforms"`
}

func newListing(commitID string, getProj ProjectProviderFunc) (*Listing, error) {
	targetCommitID, err := targettedCommitID(commitID, getProj)
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

func targettedCommitID(commitID string, getProj ProjectProviderFunc) (strfmt.UUID, error) {
	if commitID != "" {
		var cid strfmt.UUID
		err := cid.UnmarshalText([]byte(commitID))

		return cid, err
	}

	proj, fail := getProj()
	if fail != nil {
		return "", fail
	}

	cmt, fail := model.LatestCommitID(proj.Owner(), proj.Name())
	if fail != nil {
		return "", fail
	}

	if cmt == nil {
		return "", FailNoCommitID.New("error_no_commit")
	}

	return *cmt, nil
}
