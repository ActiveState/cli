package main

import (
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/rollbar/rollbar-go"
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

func (b *Bindings) Exit() {
	// This is SUPER dirty, but as of right now wails leaves us no other choice:
	// https://github.com/wailsapp/wails/issues/693
	events.WaitForEvents(1*time.Second, rollbar.Close)
	os.Exit(0)
}

func (b *Bindings) DebugMode() bool {
	args := strings.Join(os.Args, "")
	return strings.Contains(args, "wails.BuildMode=debug") || strings.Contains(args, "go-build")
}
