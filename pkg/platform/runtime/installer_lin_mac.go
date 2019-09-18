// +build !windows

package runtime

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/logging"
)

// ensure the implementation of the interface
var _ ProgressUnarchiver = &TarGzArchiveReader{}

// ProgressUnarchiver is an interface for an unarchiver with feedback about unpacking progress
type ProgressUnarchiver interface {
	UnarchiveWithProgress(string, string, func()) error
}

// TarGzArchiveReader is an extension of an TarGz archiver implementing an unarchive method with
// progress feedback
type TarGzArchiveReader struct {
	archiver.TarGz
}

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

// UnarchiveWithProgress unpacks the files from the source directory into the destination directory
// After a file is unpacked, the progressIncrement callback is called
func (ar *TarGzArchiveReader) UnarchiveWithProgress(source, destination string, progressIncrement func()) error {
	if !fileExists(destination) && ar.MkdirAll {
		err := mkdir(destination)
		if err != nil {
			return fmt.Errorf("preparing destination: %v", err)
		}
	}

	archiveStream, err := os.Open(source)
	if err != nil {
		return err
	}
	defer archiveStream.Close()

	/* We need this for the Zip file only
	stat, err := archiveStream.Stat()
	// archive size in bytes
	archiveSize := stat.Size()
	*/

	// read one file at a time from the archive
	err = ar.Open(archiveStream, 0)
	if err != nil {
		return err
	}
	// note: that this is obviously not thread-safe
	defer ar.Close()

	for {
		f, err := ar.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// calling the increment callback
		logging.Debug("Extracted %s File size: %d", f.Name(), f.Size())
		progressIncrement()
	}
	return nil
}

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

func (ar *TarGzArchiveReader) untarNext(to string) error {
	f, err := ar.Read()
	if err != nil {
		return err // don't wrap error; calling loop must break on io.EOF
	}
	header, ok := f.Header.(*tar.Header)
	if !ok {
		return fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}
	return ar.untarFile(f, filepath.Join(to, header.Name))
}

func (ar *TarGzArchiveReader) untarFile(f archiver.File, to string) error {
	// do not overwrite existing files, if configured
	if !f.IsDir() && !ar.OverwriteExisting && fileExists(to) {
		return fmt.Errorf("file already exists: %s", to)
	}

	hdr, ok := f.Header.(*tar.Header)
	if !ok {
		return fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		return mkdir(to)
	case tar.TypeReg, tar.TypeRegA, tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
		return writeNewFile(to, f, f.Mode())
	case tar.TypeSymlink:
		return writeNewSymbolicLink(to, hdr.Linkname)
	case tar.TypeLink:
		// NOTE: this is a hack that fixes an issue for choosing the correct path to the old file
		// that is being linked to. This fix will only address calls to Unarchive, not Extract and
		// is generally only known to be useful for ActiveState, at the moment.
		return writeNewHardLink(to, path.Join(path.Dir(to), path.Base(hdr.Linkname)))
	case tar.TypeXGlobalHeader:
		return nil // ignore the pax global header from git-generated tarballs
	default:
		return fmt.Errorf("%s: unknown type flag: %c", hdr.Name, hdr.Typeflag)
	}
}

// InstallerExtension is used to identify whether an artifact is one that we should care about
const InstallerExtension = ".tar.gz"

// Archiver returns the archiver to use
func Archiver() archiver.Archiver {
	return archiver.DefaultTarGz
}

// Unarchiver returns the unarchiver to use
func Unarchiver() archiver.Unarchiver {
	return archiver.DefaultTarGz
}

// UnarchiverWithProgress returns the ProgressUnarchiver to use
func UnarchiverWithProgress() *TarGzArchiveReader {
	return &TarGzArchiveReader{*archiver.DefaultTarGz}
}
