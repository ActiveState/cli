package main

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/wailsapp/wails"
)

type Bindings struct {
	update    *updater.AvailableUpdate
	changelog string
	runtime   *wails.Runtime
}

func (b *Bindings) WailsInit(runtime *wails.Runtime) error {
	b.runtime = runtime
	return nil
}

func (b *Bindings) CurrentVersion() string {
	return constants.VersionNumber
}

func (b *Bindings) AvailableVersion() string {
	return b.update.Version
}

func (b *Bindings) Changelog() string {
	return b.changelog
}

func (b *Bindings) Warning() string {
	return "This is a test warning"
}

func (b *Bindings) Exit() {
	b.runtime.Window.Close()
}

func (b *Bindings) DebugMode() bool {
	args := strings.Join(os.Args, "")
	return strings.Contains(args, "wails.BuildMode=debug") || strings.Contains(args, "go-build")
}
