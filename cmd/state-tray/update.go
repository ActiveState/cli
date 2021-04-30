package main

import (
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/getlantern/systray"
	"github.com/gobuffalo/packr"
)

const (
	iconFile            = "icon.ico"
	iconUpdateFile      = "icon-update.ico"
	updateCheckInterval = time.Second * 24
)

func superviseUpdate(mdl *model.SvcModel, notice *updateNotice) func() {
	var done chan struct{}

	go func() {
		for {
			notice.show(needsUpdate(mdl))

			select {
			case <-time.After(updateCheckInterval):
			case <-done:
				return
			}
		}
	}()

	return func() { close(done) }
}

func needsUpdate(mdl *model.SvcModel) bool {
	for i := 1; i <= 3; i++ {
		stateVersion, err := mdl.StateVersion()
		if err != nil {
			time.Sleep(time.Second * 10 * time.Duration(i))
			continue
		}
		currentVersion := constants.Version
		possibleVersion := stateVersion.State.Version

		return currentVersion != possibleVersion
	}

	logging.Error("Cannot contact servers to determine the latest state version")

	return false
}

type updateNotice struct {
	box  packr.Box
	item *systray.MenuItem
}

func (n *updateNotice) show(show bool) {
	switch {
	case true:
		n.item.Show()
		systray.SetIcon(n.box.Bytes(iconUpdateFile))
	case false:
		n.item.Hide()
		systray.SetIcon(n.box.Bytes(iconFile))
	}
}
