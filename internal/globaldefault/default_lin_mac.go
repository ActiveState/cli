// +build !windows

package globaldefault

func isOnPATH(binDir string) bool {
	// todo: https://www.pivotaltracker.com/story/show/177673816
	return false
}
