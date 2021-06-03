// +build linux darwin

package main

import (
	"os"

	"github.com/ActiveState/cli/internal/appinfo"
)

func removeSelf() error {
	return os.Remove(appinfo.StateApp().Exec())
}
