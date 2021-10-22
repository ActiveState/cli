package processor

import (
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/patrickmn/go-cache"
)

var projectIDCache *cache.Cache
var projectIDMutex *sync.Mutex

// projectID resolves the projectID from projectName and caches the result in the provided projectIDMap
func ResolveProjectID(projectNameSpace string) (string, error) {
	projectID := ""

	if projectNameSpace == "" {
		return projectID, nil
	}

	if projectIDCache == nil {
		projectIDCache = cache.New(30*time.Minute, time.Hour)
	}
	if projectIDMutex == nil {
		projectIDMutex = &sync.Mutex{}
	}

	// Lock mutex to prevent resolving the same projectName more than once
	projectIDMutex.Lock()
	defer projectIDMutex.Unlock()

	if pi, ok := projectIDCache.Get(projectNameSpace); ok {
		projectID = pi.(string)
	}

	pn, err := project.ParseNamespace(projectNameSpace)
	if err != nil {
		return projectID, errs.Wrap(err, "Failed to parse project namespace %s", projectNameSpace)
	}

	pj, err := model.FetchProjectByName(pn.Owner, pn.Project)
	if err != nil {
		return projectID, errs.Wrap(err, "Failed get project by name")
	}

	pi := string(pj.ProjectID)
	projectIDCache.Set(projectNameSpace, pi, cache.DefaultExpiration)

	return projectID, nil
}
