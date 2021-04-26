package main

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/updater"
)

type Bindings struct {
	update    *updater.AvailableUpdate
	changelog string
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

func (b *Bindings) DebugMode() bool {
	args := strings.Join(os.Args, "")
	return strings.Contains(args, "wails.BuildMode=debug") || strings.Contains(args, "go-build")
}
