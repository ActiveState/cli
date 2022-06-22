package main

import (
	"archive/zip"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/unarchiver"
	"io"
	"os"
	"path"
	"path/filepath"
)

func unpackArtifact(ua unarchiver.Unarchiver, tarballPath string, targetDir string) error {
	f, i, err := ua.PrepareUnpacking(tarballPath, targetDir)
	if err != nil {
		return errs.Wrap(err, "Prepare for unpacking failed")
	}
	defer f.Close()
	return ua.Unarchive(f, i, targetDir)
}

// Unzip unzips the file zippath and puts it in destination
func unzip(zippath string, destination string) (err error) {
	r, err := zip.OpenReader(zippath)
	if err != nil {
		return errs.Wrap(err, "Error opening zip reader")
	}
	for _, f := range r.File {
		fullname := path.Join(destination, f.Name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(fullname, f.FileInfo().Mode().Perm())
			if err != nil {
				return errs.Wrap(err, "Unable to create directory")
			}
		} else {
			err := os.MkdirAll(filepath.Dir(fullname), 0755)
			if err != nil {
				return errs.Wrap(err, "Unable to create directory II")
			}
			perms := f.FileInfo().Mode().Perm()
			out, err := os.OpenFile(fullname, os.O_CREATE|os.O_RDWR, perms)
			if err != nil {
				return errs.Wrap(err, "Unable to create output file")
			}
			defer out.Close()
			rc, err := f.Open()
			if err != nil {
				return errs.Wrap(err, "Unable to open file")
			}
			defer rc.Close()
			_, err = io.CopyN(out, rc, f.FileInfo().Size())
			if err != nil {
				return errs.Wrap(err, "Unable to CopyN")
			}

			mtime := f.FileInfo().ModTime()
			err = os.Chtimes(fullname, mtime, mtime)
			if err != nil {
				return errs.Wrap(err, "Unable to reset file time")
			}
		}
	}
	return
}

// IsZip checks to see if path is already a zip file
func isZip(path string) bool {
	r, err := zip.OpenReader(path)
	if err == nil {
		r.Close()
		return true
	}
	return false
}
