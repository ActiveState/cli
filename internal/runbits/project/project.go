package project

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// LocalProjectDoesNotExist is an error returned when a requested project is not checked out locally.
type LocalProjectDoesNotExist struct{ error }

// IsLocalProjectDoesNotExistError checks if the error is a LocalProjectDoesNotExist.
func IsLocalProjectDoesNotExistError(err error) bool {
	return errs.Matches(err, &LocalProjectDoesNotExist{})
}

// FromNamespaceLocal returns a local project (if any) that matches the given namespace.
// This is primarily used by `state use` in order to fetch a project to switch to if it already
// exists locally. The namespace may omit the owner.
func FromNamespaceLocal(ns *project.Namespaced, cfg projectfile.ConfigGetter, prompt prompt.Prompter) (*project.Project, error) {
	matchingProjects := make(map[string][]string)
	matchingNamespaces := make([]string, 0)
	for namespace, paths := range projectfile.GetProjectMapping(cfg) {
		if len(paths) == 0 {
			continue
		}
		var namespaced project.Namespaced
		err := namespaced.Set(namespace)
		if err != nil {
			logging.Debug("Cannot parse namespace: %v") // should not happen since this is stored
			continue
		}
		if (!ns.AllowOmitOwner && strings.ToLower(namespaced.String()) == strings.ToLower(ns.String())) ||
			(ns.AllowOmitOwner && strings.ToLower(namespaced.Project) == strings.ToLower(ns.Project)) {
			matchingProjects[namespace] = paths
			matchingNamespaces = append(matchingNamespaces, namespace)
		}
	}

	if len(matchingProjects) > 0 {
		var err error

		sort.Strings(matchingNamespaces)
		namespace := matchingNamespaces[0]
		if len(matchingProjects) > 1 {
			namespace, err = prompt.Select(
				"",
				locale.Tl("project_select_namespace", "Multiple projects with that name were found. Please select one."),
				matchingNamespaces,
				&namespace)
			if err != nil {
				return nil, locale.WrapError(err, "err_project_select_namespace", "Error selecting project")
			}
		}

		paths, exists := matchingProjects[namespace]
		if !exists {
			return nil, errs.New("Selected project not mapped to a namespace") // programmer error
		}

		sort.Strings(paths)
		path := paths[0]
		if len(paths) > 1 {
			path, err = prompt.Select(
				"",
				locale.Tl("project_select_path", "Multiple project paths for the selected project were found. Please select one."),
				paths,
				&path)
			if err != nil {
				return nil, locale.WrapError(err, "err_project_select_path", "Error selecting project path")
			}
		}

		return project.FromPath(path)
	}

	projectsDir, err := storage.ProjectsDir(cfg)
	if err != nil {
		return nil, locale.WrapError(err, "err_cannot_determine_projects_dir")
	}
	projectDir := filepath.Join(projectsDir, ns.Project)
	return nil, &LocalProjectDoesNotExist{
		locale.NewInputError("err_local_project_not_checked_out", "", ns.Project, projectDir),
	}
}
