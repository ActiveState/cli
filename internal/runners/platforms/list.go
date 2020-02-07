package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
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

	fetcher, err := newFetchByCommitID("")
	if err != nil {
		return nil, err
	}

	return newListing(fetcher)
}

// Listing represents the output data of a listing.
type Listing struct {
	Platforms []*Platform `json:"platforms"`
}

func newListing(f fetcher) (*Listing, error) {
	platforms, err := f.FetchPlatforms()
	if err != nil {
		return nil, err
	}

	listing := Listing{
		Platforms: makePlatformsFromModelPlatforms(platforms),
	}

	return &listing, nil
}
