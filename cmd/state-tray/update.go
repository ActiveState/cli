package main

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/getlantern/systray"
)

const (
	updateCheckInterval = time.Hour
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
	availableUpdate, err := mdl.CheckUpdate()
	if err != nil {
		logging.Errorf("Cannot determine the latest state version: %s", errs.Join(err, ","))
		return false
	}

	return availableUpdate != nil
}

type updateNotice struct {
	item *systray.MenuItem
}

func (n *updateNotice) show(show bool) {
	switch show {
	case true:
		n.item.Show()
		systray.SetIcon(iconUpdateFile)
	case false:
		n.item.Hide()
		systray.SetIcon(iconFile)
	}
}
