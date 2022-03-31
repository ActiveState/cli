//go:build !windows
// +build !windows

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

func (i *Installer) cleanInstallPath() error {
	if fileutils.DirExists(i.path) {
		files, err := ioutil.ReadDir(i.path)
		if err != nil {
			return errs.Wrap(err, "Could not installation directory: %s", i.path)
		}

		for _, file := range files {
			fname := strings.ToLower(file.Name())
			if isStateExecutable(fname) {
				err = os.Remove(filepath.Join(i.path, fname))
				if err != nil {
					return errs.Wrap(err, "Could not remove")
				}
			}
		}

		return nil
	}
}
