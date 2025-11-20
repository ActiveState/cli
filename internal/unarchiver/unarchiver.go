package unarchiver

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mholt/archives"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

type Unarchiver struct {
	archives.Extraction
}

func NewTarGz() Unarchiver {
	return Unarchiver{archives.CompressedArchive{
		Compression: archives.Gz{},
		Extraction:  archives.Tar{},
	}}
}

func NewZip() Unarchiver {
	return Unarchiver{
		archives.CompressedArchive{
			Extraction: archives.Zip{},
		},
	}
}

// PrepareUnpacking prepares the destination directory and the archive for unpacking
// Returns the opened file
func (ua *Unarchiver) PrepareUnpacking(source, destination string) (archiveFile *os.File, err error) {

	if !fileutils.DirExists(destination) {
		err := mkdir(destination)
		if err != nil {
			return nil, fmt.Errorf("preparing destination: %v", err)
		}
	}

	archiveFile, err = os.Open(source)
	if err != nil {
		return nil, err
	}

	return archiveFile, nil

}

// Unarchive unarchives an archive file and unpacks it in `destination`
func (ua *Unarchiver) Unarchive(archiveStream io.Reader, destination string) error {
	ctx := context.Background()
	err := ua.Extract(ctx, archiveStream, func(_ context.Context, file archives.FileInfo) error {
		path := filepath.Join(destination, file.NameInArchive)

		if file.IsDir() {
			return mkdir(path)
		}

		if file.LinkTarget != "" {
			if file.Mode()&os.ModeSymlink != 0 {
				return writeNewSymbolicLink(path, file.LinkTarget)
			}
			target := filepath.Join(destination, file.LinkTarget)
			return writeNewHardLink(path, target)
		}

		f, err := file.Open()
		if err != nil {
			return err
		}
		defer f.Close()

		return writeNewFile(path, f, file.Mode())
	})
	if err != nil {
		return errs.Wrap(err, "Unable to extract files")
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

	// The unarchiving process is unordered, and a hardlinked file's target may not yet exist.
	// Create it. writeNewFile() will overwrite it later, which is okay.
	if !fileExists(target) {
		err = os.MkdirAll(filepath.Dir(target), 0755)
		if err != nil {
			return fmt.Errorf("%s: making directory for file: %v", target, err)
		}
		f, err := os.Create(target)
		if err != nil {
			return fmt.Errorf("%s: creating target file: %v", target, err)
		}
		f.Close()
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
