//go:build linux || windows
// +build linux windows

package installmgr

func RemoveSystemFiles(_ string) error {
	return nil
}
