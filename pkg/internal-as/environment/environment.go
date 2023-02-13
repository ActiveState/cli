// Package environment exposes some of the environment internal package.
package environment

import "github.com/ActiveState/cli/internal/environment"

// GetRootPath returns the root path of the library we're under
func GetRootPath() (string, error) {
	return environment.GetRootPath()
}

func GetRootPathUnsafe() string {
	return environment.GetRootPathUnsafe()
}
