package findproject

import (
	"errors"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// LocalProjectDoesNotExist is an error returned when a requested project is not checked out locally.
type LocalProjectDoesNotExist struct{ *locale.LocalizedError }

// IsLocalProjectDoesNotExistError checks if the error is a LocalProjectDoesNotExist.
func IsLocalProjectDoesNotExistError(err error) bool {
	var errLocalProjectDoesNotExist *LocalProjectDoesNotExist
	return errors.As(err, &errLocalProjectDoesNotExist)
}

func FromInputByPriority(path string, ns *project.Namespaced, cfg projectfile.ConfigGetter, prompt prompt.Prompter) (*project.Project, error) {
	// Priority #1 - PATH
	if path != "" {
		return FromPath(path, ns)
	}

	// Priority #2 - Namespace
	if ns != nil && ns.IsValid() {
		return FromNamespaceLocal(ns, cfg, prompt)
	}

	// Priority #3 - Env
	pj, err := project.FromEnv()
	if err != nil {
		return nil, locale.WrapError(err, "err_project_fromenv")
	}

	return pj, nil
}

func FromPath(path string, ns *project.Namespaced) (*project.Project, error) {
	pj, err := project.FromPath(path)
	if err != nil {
		return nil, &LocalProjectDoesNotExist{locale.WrapInputError(err, "err_project_frompath_notexist", "", path)}
	}

	if ns != nil && ns.IsValid() && ((ns.Owner != "" && pj.Namespace().Owner != ns.Owner) || pj.Namespace().Project != ns.Project) {
		return nil, locale.WrapInputError(err, "err_project_namespace_missmatch", "", path, ns.String())
	}
	return pj, nil
}

// FromNamespaceLocal returns a local project (if any) that matches the given namespace (or the
// project in the current working directory if namespace was not given).
// This is primarily used by `state use` in order to fetch a project to switch to if it already
// exists locally. The namespace may omit the owner.
func FromNamespaceLocal(ns *project.Namespaced, cfg projectfile.ConfigGetter, prompt prompt.Prompter) (*project.Project, error) {
	if ns == nil || !ns.IsValid() {
		root, err := osutils.Getwd()
		if err != nil {
			return nil, locale.WrapInputError(err, "Unable to determine current working directory. Please specify a project to use.")
		}
		return project.FromPath(root)
	}

	// Get the stale project mapping early as GetProjectMapping will clean stale projects
	staleProjects := projectfile.GetStaleProjectMapping(cfg)

	matchingProjects := make(map[string][]string)
	matchingNamespaces := make([]string, 0)
	for namespace, paths := range projectfile.GetProjectMapping(cfg) {
		if len(paths) == 0 {
			continue
		}
		namespaced, err := project.ParseNamespace(namespace)
		if err != nil {
			logging.Debug("Cannot parse namespace: %v") // should not happen since this is stored
			continue
		}
		if !ns.AllowOmitOwner && strings.EqualFold(strings.ToLower(namespaced.String()), strings.ToLower(ns.String())) ||
			(ns.AllowOmitOwner && strings.EqualFold(strings.ToLower(namespaced.Project), strings.ToLower(ns.Project))) {
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
				&namespace,
				nil)
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
				&path,
				nil)
			if err != nil {
				return nil, locale.WrapError(err, "err_project_select_path", "Error selecting project path")
			}
		}

		return project.FromPath(path)
	}

	for namespace, paths := range staleProjects {
		namespaced, err := project.ParseNamespace(namespace)
		if err != nil {
			logging.Debug("Cannot parse namespace: %v") // should not happen since this is stored
			continue
		}

		if !ns.AllowOmitOwner && strings.EqualFold(strings.ToLower(namespaced.String()), strings.ToLower(ns.String())) ||
			(ns.AllowOmitOwner && strings.EqualFold(strings.ToLower(namespaced.Project), strings.ToLower(ns.Project))) && len(paths) > 0 {
			return nil, &LocalProjectDoesNotExist{
				locale.NewInputError("err_findproject_notfound", "", ns.Project, paths[0]),
			}
		}
	}

	return nil, &LocalProjectDoesNotExist{
		locale.NewInputError("err_local_project_not_checked_out", "", ns.Project),
	}
}
