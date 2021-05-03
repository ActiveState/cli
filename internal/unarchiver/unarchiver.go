// Package unarchiver provides a method to unarchive tar.gz or zip archives with progress bar feedback
// Currently, this implementation copies a lot of methods that are internal to the ActiveState/archiver dependency.
package unarchiver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/archiver"

	"github.com/ActiveState/cli/internal/fileutils"
)

// SingleUnarchiver is an interface for an unarchiver that can unpack the next file
// It extends the existing archiver.Reader with a method to extract a single file from the archive
type SingleUnarchiver interface {
	archiver.Reader

	// ExtractNext extracts the next file in the archive
	ExtractNext(destination string) (f archiver.File, err error)

	// CheckExt checks that the file extension is appropriate for the archive
	CheckExt(archiveName string) error

	// Ext returns a valid file name extension for this archiver
	Ext() string
}

// ExtractNotifier gets called when a new file has been extracted from the archive
type ExtractNotifier func(fileName string, size int64, isDir bool)

// Unarchiver wraps an implementation of an unarchiver that can unpack one file at a time.
type Unarchiver struct {
	// wraps a struct that can unpack one file at a time.
	impl SingleUnarchiver

	notifier ExtractNotifier
}

func (ua *Unarchiver) Ext() string {
	return ua.impl.Ext()
}

// SetNotifier sets the notification function to be called after extracting a file
func (ua *Unarchiver) SetNotifier(cb ExtractNotifier) {
	ua.notifier = cb
}

// PrepareUnpacking prepares the destination directory and the archive for unpacking
// Returns the opened file and its size
func (ua *Unarchiver) PrepareUnpacking(source, destination string) (archiveFile *os.File, fileSize int64, err error) {

	if !fileutils.DirExists(destination) {
		err := mkdir(destination)
		if err != nil {
			return nil, 0, fmt.Errorf("preparing destination: %v", err)
		}
	}

	archiveFile, err = os.Open(source)
	if err != nil {
		return nil, 0, err
	}

	fileInfo, err := archiveFile.Stat()
	if err != nil {
		archiveFile.Close()
		return nil, 0, fmt.Errorf("statting source file: %v", err)
	}

	return archiveFile, fileInfo.Size(), nil

}

// CheckExt checks that the file extension is appropriate for the given unarchiver
func (ua *Unarchiver) CheckExt(archiveName string) error {
	return ua.impl.CheckExt(archiveName)
}

// Unarchive unarchives an archive file ` and unpacks it in `destination`
func (ua *Unarchiver) Unarchive(archiveStream io.Reader, archiveSize int64, destination string) (err error) {
	// impl is the actual implementation of the unarchiver (tar.gz or zip)
	impl := ua.impl

	// read one file at a time from the archive
	err = impl.Open(archiveStream, archiveSize)
	if err != nil {
		return
	}
	// note: that this is obviously not thread-safe
	defer impl.Close()

	for {
		// extract one file at a time
		var f archiver.File
		f, err = impl.ExtractNext(destination)
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}

		//logging.Debug("Extracted %s File size: %d", f.Name(), f.Size())
		ua.notifier(f.Name(), f.Size(), f.IsDir())
	}

	return nil
}

// the following files are just copied from the ActiveState/archiver repository
// so we can use them in our extensions

func writeNewFile(fpath string, in io.Reader, fm os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	out, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("%s: creating new file: %v", fpath, err)
	}
	defer out.Close()

	err = out.Chmod(fm)
	if err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("%s: changing file mode: %v", fpath, err)
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("%s: writing file: %v", fpath, err)
	}
	return nil
}

func writeNewSymbolicLink(fpath string, target string) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	err = os.Symlink(target, fpath)
	if err != nil {
		return fmt.Errorf("%s: making symbolic link for: %v", fpath, err)
	}

	return nil
}

func writeNewHardLink(fpath string, target string) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	err = os.Link(target, fpath)
	if err != nil {
		return fmt.Errorf("%s: making hard link for: %v", fpath, err)
	}

	return nil
}

// ensure the implementation of the interface
func fileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func mkdir(dirPath string) error {
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory: %v", dirPath, err)
	}
	return nil
}
