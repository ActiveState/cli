// +build linux darwin

package config

// getLockFile returns the config file on Linux and Darwin, as we using it as a lockfile
func (i *Instance) getLockFile() string {
	return i.getConfigFile()
}

// cleanLockFile does nothing on Linux and Darwin, as we do not create a separate file for locking
func (i *Instance) cleanLockFile() {
}
