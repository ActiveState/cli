//go:build linux || darwin
// +build linux darwin

package updater

func checkAdmin() error {
	return nil
}
