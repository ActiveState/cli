package projectcache

import (
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/patrickmn/go-cache"
)

type ID struct {
	projectIDCache *cache.Cache
	projectIDMutex *sync.Mutex // used to synchronize API calls resolving the projectID
}

func NewID() *ID {
	return &ID{
		cache.New(30*time.Minute, time.Hour),
		&sync.Mutex{},
	}
}

// FromNamespace resolves the projectID from projectName and caches the result
func (i *ID) FromNamespace(projectNameSpace string) (string, error) {
	// Lock mutex to prevent resolving the same projectName more than once
	i.projectIDMutex.Lock()
	defer i.projectIDMutex.Unlock()

	if pi, ok := i.projectIDCache.Get(projectNameSpace); ok {
		return pi.(string), nil
	}

	pn, err := project.ParseNamespace(projectNameSpace)
	if err != nil {
		return "", errs.Wrap(err, "Failed to parse project namespace %s", projectNameSpace)
	}

	pj, err := model.FetchProjectByName(pn.Owner, pn.Project)
	if err != nil {
		return "", errs.Wrap(err, "Failed get project by name")
	}

	pi := string(pj.ProjectID)
	i.projectIDCache.Set(projectNameSpace, pi, cache.DefaultExpiration)

	return string(pj.ProjectID), nil
}
