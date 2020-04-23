// +build windows

package deploy

import "golang.org/x/sys/windows/registry"

func runSymlinkTests() bool {
	return isWindowsDeveloperModeActive()
}

func isWindowsDeveloperModeActive() bool {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\AppModelUnlock", registry.READ)
	if err != nil {
		return false
	}

	val, _, err := key.GetIntegerValue("AllowDevelopmentWithoutDevLicense")
	if err != nil {
		return false
	}

	return val != 0
}
