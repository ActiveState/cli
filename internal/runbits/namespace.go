package runbits

import (
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/thoas/go-funk"
)

type ConfigAble interface {
	Set(key string, value interface{})
	GetString(key string) string
	GetStringSlice(key string) []string
}

// AvailableProjectPaths returns the paths of all projects associated with the namespace
func AvailableProjectPaths(c ConfigAble, namespace string) []string {
	key := fmt.Sprintf("project_%s", namespace)
	paths := c.GetStringSlice(key)
	paths = funk.FilterString(paths, func(path string) bool {
		return fileutils.FileExists(filepath.Join(path, constants.ConfigFileName))
	})
	paths = funk.UniqString(paths)
	c.Set(key, paths)
	return paths
}
