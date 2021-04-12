//+build linux

package clean

func (u *Uninstall) removeInstall() error {
	return u.removeInstallDir()
}
