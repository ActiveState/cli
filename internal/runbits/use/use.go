package use

import (
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func GetLocalProjectPath(ns *project.Namespaced, cfg *config.Instance) string {
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
			return paths[0] // just pick the first one
		}
	}
	return ""
}
