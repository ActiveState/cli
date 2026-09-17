package unarchiver

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mholt/archives"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

type Unarchiver struct {
	archives.Extraction

	untrusted bool
}

// Option configures an Unarchiver.
type Option func(*Unarchiver)

// WithUntrustedSource marks the archive as coming from an untrusted source, so
// every extracted path, symlink target, and hardlink target is confined under
// the destination root and anything that would escape aborts extraction. Use it
// for untrusted archives such as private ingredient wheels.
//
// It is off by default: trusted Platform artifacts may legitimately contain
// absolute symlinks (for example into /usr/share), which would otherwise be
// rejected.
func WithUntrustedSource() Option {
	return func(ua *Unarchiver) { ua.untrusted = true }
}

func NewTarGz(opts ...Option) Unarchiver {
	return newUnarchiver(archives.CompressedArchive{
		Compression: archives.Gz{},
		Extraction:  archives.Tar{},
	}, opts)
}

func NewZip(opts ...Option) Unarchiver {
	return newUnarchiver(archives.Zip{}, opts)
}

func newUnarchiver(extraction archives.Extraction, opts []Option) Unarchiver {
	ua := Unarchiver{Extraction: extraction}
	for _, opt := range opts {
		opt(&ua)
	}
	return ua
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

// Unarchive unarchives an archive file and unpacks it in `destination`. For an
// archive from an untrusted source (see WithUntrustedSource), every entry path,
// symlink target, and hardlink target is confined under destination and anything
// that would escape aborts extraction; otherwise paths are trusted as-is.
func (ua *Unarchiver) Unarchive(archiveStream io.Reader, destination string) error {
	root := filepath.Clean(destination)
	ctx := context.Background()
	err := ua.Extract(ctx, archiveStream, func(_ context.Context, file archives.FileInfo) error {
		path := filepath.Join(root, file.NameInArchive)
		if ua.untrusted && !isContainedPath(root, path) {
			return errs.New("entry %q escapes the extraction root", file.NameInArchive)
		}

		if file.IsDir() {
			if err := mkdir(path); err != nil {
				return errs.Wrap(err, "could not create directory")
			}
			return nil
		}

		if file.LinkTarget != "" {
			if file.Mode()&os.ModeSymlink != 0 {
				if ua.untrusted {
					if hasAbsoluteTarget(file.LinkTarget) {
						return errs.New("symlink target %q is absolute", file.LinkTarget)
					}
					resolved := filepath.Join(filepath.Dir(path), file.LinkTarget)
					if !isContainedPath(root, resolved) {
						return errs.New("symlink target %q escapes the extraction root", file.LinkTarget)
					}
				}
				if err := writeNewSymbolicLink(path, file.LinkTarget); err != nil {
					return errs.Wrap(err, "could not write symlink")
				}
				return nil
			}
			target := filepath.Join(root, file.LinkTarget)
			if ua.untrusted && !isContainedPath(root, target) {
				return errs.New("hardlink target %q escapes the extraction root", file.LinkTarget)
			}
			if err := writeNewHardLink(path, target); err != nil {
				return errs.Wrap(err, "could not write hardlink")
			}
			return nil
		}

		f, err := file.Open()
		if err != nil {
			return errs.Wrap(err, "could not open archived file")
		}
		defer f.Close()

		if err := writeNewFile(path, f, file.Mode()); err != nil {
			return errs.Wrap(err, "could not write file")
		}
		return nil
	})
	if err != nil {
		return errs.Wrap(err, "Unable to extract files")
	}

	return nil
}

// isContainedPath reports whether path is at or under root. Both are expected to
// be cleaned (filepath.Join cleans its result).
func isContainedPath(root, path string) bool {
	return path == root || strings.HasPrefix(path, root+string(os.PathSeparator))
}

// hasAbsoluteTarget reports whether an archive link target is rooted on any
// platform. filepath.IsAbs alone is host-specific, so a Unix-style "/etc/passwd"
// reads as relative on Windows; this checks for a leading separator (Unix "/" or
// Windows "\") or a volume name ("C:") instead.
func hasAbsoluteTarget(target string) bool {
	return target != "" && (target[0] == '/' || target[0] == '\\' || filepath.VolumeName(target) != "")
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
