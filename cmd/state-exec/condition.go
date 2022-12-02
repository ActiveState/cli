package main

import "os"

// onCI is copied from the internal/condition package (to minimize depdencies).
func onCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != ""
}
