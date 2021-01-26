package project

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var projectMapMutex = &sync.Mutex{}

type ConfigGetter interface {
	GetStringMapStringSlice(key string) map[string][]string
	AllKeys() []string
	GetStringSlice(string) []string
	Set(string, interface{})
}

func GetProjectMapping(config ConfigGetter) map[string][]string {
	addDeprecatedProjectMappings(config)
	CleanProjectMapping(config)
	projects := config.GetStringMapStringSlice(projectfile.LocalProjectsConfigKey)
	if projects == nil {
		return map[string][]string{}
	}
	return projects
}

func GetProjectNameForPath(config ConfigGetter, projectPath string) string {
	projects := GetProjectMapping(config)

	for name, paths := range projects {
		if name == "/" {
			continue
		}
		for _, path := range paths {
			if isEqual, err := fileutils.PathsEqual(projectPath, path); isEqual {
				if err != nil {
					logging.Debug("Failed to compare paths %s and %s", projectPath, path)
				}
				return name
			}
		}
	}
	return ""
}

func addDeprecatedProjectMappings(config ConfigGetter) {
	projects := config.GetStringMapStringSlice(projectfile.LocalProjectsConfigKey)
	keys := funk.FilterString(config.AllKeys(), func(v string) bool {
		return strings.HasPrefix(v, "project_")
	})

	if len(keys) == 0 {
		return
	}

	for _, key := range keys {
		namespace := strings.TrimPrefix(key, "project_")
		newPaths := projects[namespace]
		paths := config.GetStringSlice(key)
		projects[namespace] = funk.UniqString(append(newPaths, paths...))
		config.Set(key, nil)
	}

	config.Set(projectfile.LocalProjectsConfigKey, projects)
}

// GetProjectPaths returns the paths of all projects associated with the namespace
func GetProjectPaths(config ConfigGetter, namespace string) []string {
	projects := GetProjectMapping(config)

	// match case-insensitively
	var paths []string
	for key, value := range projects {
		if strings.ToLower(key) == strings.ToLower(namespace) {
			paths = append(paths, value...)
		}
	}

	return paths
}

// storeProjectMapping associates the namespace with the project
// path in the config
func storeProjectMapping(cfg ConfigGetter, namespace, projectPath string) {
	projectMapMutex.Lock()
	defer projectMapMutex.Unlock()

	projectPath = filepath.Clean(projectPath)

	projects := cfg.GetStringMapStringSlice(projectfile.LocalProjectsConfigKey)
	if projects == nil {
		projects = make(map[string][]string)
	}

	paths := projects[namespace]
	if paths == nil {
		paths = make([]string, 0)
	}

	if !funk.Contains(paths, projectPath) {
		paths = append(paths, projectPath)
	}

	projects[namespace] = paths
	cfg.Set(projectfile.LocalProjectsConfigKey, projects)
}

// CleanProjectMapping removes projects that no longer exist
// on a user's filesystem from the projects config entry
func CleanProjectMapping(cfg ConfigGetter) {
	projects := cfg.GetStringMapStringSlice(projectfile.LocalProjectsConfigKey)
	seen := map[string]bool{}

	for namespace, paths := range projects {
		for i, path := range paths {
			if !fileutils.DirExists(path) {
				projects[namespace] = sliceutils.RemoveFromStrings(projects[namespace], i)
			}
		}
		if ok, _ := seen[strings.ToLower(namespace)]; ok || len(projects[namespace]) == 0 {
			delete(projects, namespace)
			continue
		}
		seen[strings.ToLower(namespace)] = true
	}

	cfg.Set("projects", projects)
}
