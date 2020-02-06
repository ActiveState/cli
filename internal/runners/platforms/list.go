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

	fetcher := &fetch{}

	return newListing(fetcher)
}

// Listing represents the output data of a listing.
type Listing struct {
	Platforms []*Platform `json:"platforms"`
}

func newListing(fetcher Fetcher) (*Listing, error) {
	platforms, fail := fetcher.FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	var listing Listing
	for _, platform := range platforms {
		var p Platform
		if platform.DisplayName != nil {
			p.Name = *platform.DisplayName
		}
		listing.Platforms = append(listing.Platforms, &p)
	}

	return &listing, nil
}
