package platforms

import (
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// Fetcher describes the behavior needed to obtain platforms.
type Fetcher interface {
	FetchPlatforms() ([]*model.Platform, error)
}

type fetch struct{}

// FetchPlatforms implements the Fetcher interface.
func (f *fetch) FetchPlatforms() ([]*model.Platform, error) {
	platforms, fail := model.FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	return platforms, nil
}

type fetchByCommitID struct {
	commitID strfmt.UUID
}

func newFetchByCommitID(commitID string) (*fetchByCommitID, error) {
	if commitID == "" {
		proj := project.Get()
		cmt, fail := model.LatestCommitID(proj.Owner(), proj.Name())
		if fail != nil {
			return nil, fail
		}
		commitID = cmt.String()
	}

	var cid strfmt.UUID
	if err := cid.UnmarshalText([]byte(commitID)); err != nil {
		return nil, err
	}

	fetch := fetchByCommitID{
		commitID: cid,
	}

	return &fetch, nil
}

func (f *fetchByCommitID) FetchPlatforms() ([]*model.Platform, error) {
	platforms, fail := model.FetchPlatformsForCommit(f.commitID)
	if fail != nil {
		return nil, fail
	}

	return platforms, nil
}
