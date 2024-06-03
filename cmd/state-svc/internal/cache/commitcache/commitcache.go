package commitcache

import (
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/go-openapi/strfmt"
)

type Cache struct {
	mutex   *sync.Mutex
	bpModel *bpModel.BuildPlanner
	Commits map[string]entry
}

type entry struct {
	time   time.Time
	commit *response.Commit
}

// MaxCommits is the maximum number of commits that we should store in the cache.
// If we exceed this number, we will start to delete the oldest entries.
const MaxCommits = 50

func New(m *bpModel.BuildPlanner) *Cache {
	return &Cache{
		mutex:   &sync.Mutex{},
		bpModel: m,
		Commits: make(map[string]entry),
	}
}

func (c *Cache) Get(owner, project, commitID string) (*response.Commit, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	id := owner + project + commitID
	if v, ok := c.Commits[id]; ok {
		return v.commit, nil
	}

	commit, err := c.bpModel.FetchRawCommit(strfmt.UUID(commitID), owner, project, nil)
	if err != nil {
		return nil, errs.Wrap(err, "Could not fetch commit")
	}

	// Delete oldest entry
	if len(c.Commits) > MaxCommits {
		var oldestK string
		var oldest *entry
		for k, v := range c.Commits {
			if oldest == nil || v.time.Before(oldest.time) {
				oldest = &v
				oldestK = k
			}
		}
		delete(c.Commits, oldestK)
	}

	c.Commits[id] = entry{
		time:   time.Now(),
		commit: commit,
	}

	return commit, nil
}
