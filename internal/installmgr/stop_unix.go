//go:build linux || darwin
// +build linux darwin

package installmgr

func isAccessDeniedError(err error) bool {
	return false
}
