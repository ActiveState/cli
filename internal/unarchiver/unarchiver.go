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
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
)

// SingleUnarchiver is an interface for an unarchiver that can unpack the next file
// It extends the existing archiver.Reader with a method to extract a single file from the archive
type SingleUnarchiver interface {
	archiver.Reader

	// ExtractNext extracts the next file in the archive
	ExtractNext(destination string) (f archiver.File, err error)
}

// Unarchiver wraps an implementation of an unarchiver that can unpack one file at a time.
type Unarchiver struct {
	// wraps a struct that can unpack one file at a time.
	impl SingleUnarchiver
}

// UnarchiveWithProgress unarchives an archive file `source` and unpacks it in `destionation`
// Progress is reported with a progress bar added to `p`
func (ua *Unarchiver) UnarchiveWithProgress(source, destination string, p *progress.Progress) (err error) {
	if !fileExists(destination) {
		err := mkdir(destination)
		if err != nil {
			return fmt.Errorf("preparing destination: %v", err)
		}
	}

	archiveFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer archiveFile.Close()

	fileInfo, err := archiveFile.Stat()
	if err != nil {
		return fmt.Errorf("statting source file: %v", err)
	}

	archiveSizeIn := fileInfo.Size()

	// Add the progress bar for unpacking
	pb := p.AddUnpackBar(archiveSizeIn)

	// and wrap the stream, such that we automatically report progress while reading bytes
	wrappedStream := pb.NewProxyReader(archiveFile)

	// impl is the actual implementation of the unarchiver (tar.gz or zip)
	impl := ua.impl

	// read one file at a time from the archive
	err = impl.Open(wrappedStream, archiveSizeIn)
	if err != nil {
		return err
	}
	// note: that this is obviously not thread-safe
	defer impl.Close()

	for {
		// extract one file at a time
		f, err := impl.ExtractNext(destination)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		logging.Debug("Extracted %s File size: %d", f.Name(), f.Size())
	}

	// Set the progress bar to complete state
	pb.Complete()

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
