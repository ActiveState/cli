package main

import (
	"time"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/getlantern/systray"
	"github.com/gobuffalo/packr"
)

const (
	iconFile         = "icon.ico"
	iconUpdateFile   = "icon-update.ico"
	iconUpdatingFile = "icon-updating.ico"
)

func superviseUpdate(box packr.Box, updMenuItem *systray.MenuItem) func() {
	var done chan struct{}

	go func() {
		for {
			ico := iconFile
			hideFn := updMenuItem.Hide
			if needsUpdate() {
				ico = iconUpdateFile
				hideFn = updMenuItem.Show
			}
			if isUpdating() {
				ico = iconUpdatingFile
				hideFn = updMenuItem.Hide
			}
			systray.SetIcon(box.Bytes(ico))
			hideFn()
			time.Sleep(time.Second * 3)

			select {
			case <-time.After(time.Second * 3):
			case <-done:
				return
			}
		}
	}()

	return func() { done <- struct{}{} }
}

func needsUpdate() bool {
	return fileutils.FileExists("/home/devx1/update")
}

func isUpdating() bool {
	return fileutils.FileExists("/home/devx1/updating")
}
