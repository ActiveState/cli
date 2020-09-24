// +build !windows

package prepare

func updateEnvironment(filepath string) error {
	return updatePath(filepath)
}
