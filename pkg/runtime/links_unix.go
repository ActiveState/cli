//go:build linux || darwin
// +build linux darwin

package runtime

func supportsHardLinks(path string) bool {
	return true
}
