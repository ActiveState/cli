package main

import "os"

// inActiveStateCI is copied from the internal/condition package (to minimize dependencies).
func inActiveStateCI() bool {
	return os.Getenv("ACTIVESTATE_CI") == "true"
}
