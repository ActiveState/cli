package autostart

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/osutils/autostart"
)

func New() *autostart.App {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	return autostart.New("activestate-desktop", filepath.Join(filepath.Dir(os.Args[0]), "state-tray"+suffix))
}
