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

func superviseIcon(box packr.Box) func() {
	var done chan struct{}

	go func() {
		for {
			ico := iconFile
			if needsUpdate() {
				ico = iconUpdateFile
			}
			if isUpdating() {
				ico = iconUpdatingFile
			}
			systray.SetIcon(box.Bytes(ico))
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
