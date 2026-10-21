package smartlink

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

// LinkContents will link the contents of src to desc
func LinkContents(src, dest string) error {
	if !fileutils.DirExists(src) {
		return errs.New("src dir does not exist: %s", src)
	}
	if err := fileutils.MkdirUnlessExists(dest); err != nil {
		return errs.Wrap(err, "Could not create dir: %s", dest)
	}

	var err error
	src, dest, err = resolvePaths(src, dest)
	if err != nil {
		return errs.Wrap(err, "Could not resolve src and dest paths")
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return errs.Wrap(err, "Reading dir %s failed", src)
	}
	// src and dest are already resolved above, so recurse via link() which does not
	// re-resolve each entry. On Windows resolving a path is a syscall (GetLongPathName),
	// so resolving once per tree instead of once per file is a large speed-up when
	// installing runtimes that contain many files.
	for _, entry := range entries {
		if err := link(filepath.Join(src, entry.Name()), filepath.Join(dest, entry.Name()), entry.Type()); err != nil {
			return errs.Wrap(err, "Link failed")
		}
	}

	return nil
}

// Link creates a link from src to target. MS decided to support Symlinks but only if you opt into developer mode (go figure),
// which we cannot reasonably force on our users. So on Windows we will instead create dirs and hardlinks.
func Link(src, dest string) error {
	resolvedDest, err := fileutils.ResolveUniquePath(dest)
	if err != nil {
		return errs.Wrap(err, "Could not resolve dest path")
	}

	// Resolve the parent of src rather than src itself and rejoin the base name. ResolveUniquePath
	// dereferences symlinks, but link() needs to see whether the leaf is itself a symlink, so we must
	// not dereference it here. For a non-symlink leaf this is equivalent to resolving src directly.
	srcParent, err := fileutils.ResolveUniquePath(filepath.Dir(src))
	if err != nil {
		return errs.Wrap(err, "Could not resolve src path")
	}
	src = filepath.Join(srcParent, filepath.Base(src))

	info, err := os.Lstat(src)
	if err != nil {
		return errs.Wrap(err, "Could not stat src path")
	}
	destDir := filepath.Dir(resolvedDest)
	if err := fileutils.MkdirUnlessExists(destDir); err != nil {
		return errs.Wrap(err, "could not create directory %s", destDir)
	}

	return link(src, resolvedDest, info.Mode().Type())
}

// link recursively links src into dest. Unlike Link, it assumes src and dest are already resolved to
// unique paths and does NOT re-resolve them for every entry it visits. Path resolution is a syscall
// per path on Windows (GetLongPathName), so resolving once per tree rather than once per file makes
// installing runtimes with many files dramatically faster. Descendant paths are constructed by joining
// already-resolved parents with real (long) entry names, so they need no further resolution.
//
// typ is src's type as reported by Lstat (or by the directory listing it came from), and dest's parent
// directory must already exist. Together these mean a regular file costs a single syscall: the link itself.
func link(src, dest string, typ fs.FileMode) error {
	if typ&fs.ModeSymlink != 0 {
		if fileutils.IsDir(src) {
			// If src is a symlink, the resolved src is no longer a symlink and could point
			// to a parent directory, resulting in a recursive directory structure.
			// Avoid any potential problems by simply linking the symlink to the target.
			// Links to directories are okay on Linux and macOS, but will fail on Windows.
			// If we ever get here on Windows, the artifact being deployed is bad and there's nothing we
			// can do about it except receive the report from Rollbar and report it internally.
			if runtime.GOOS == "windows" {
				return errs.New("src is a directory, not a file: %s", src)
			}
			return linkFile(src, dest)
		}

		// A symlink whose target is a file: link to the resolved target to preserve pre-existing behavior.
		resolvedSrc, err := fileutils.ResolveUniquePath(src)
		if err != nil {
			return errs.Wrap(err, "could not resolve src path %s", src)
		}
		src = resolvedSrc
		typ = 0
	}

	if typ.IsDir() {
		if err := fileutils.Mkdir(dest); err != nil {
			return errs.Wrap(err, "could not create directory %s", dest)
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return errs.Wrap(err, "could not read directory %s", src)
		}
		for _, entry := range entries {
			if err := link(filepath.Join(src, entry.Name()), filepath.Join(dest, entry.Name()), entry.Type()); err != nil {
				return errs.Wrap(err, "sub link failed")
			}
		}
		return nil
	}

	if err := linkFile(src, dest); err != nil {
		// Multiple artifacts can supply the same file. We do not have a better solution for this at the moment other
		// than favouring the first one encountered. Artifacts may be linked concurrently (the caller must not run
		// copies at the same time; see depot.DeployViaLink), so whichever links the file first wins.
		if os.IsExist(err) {
			logging.Warning("Skipping linking '%s' to '%s' as it already exists", src, dest)
			return nil
		}
		return errs.Wrap(err, "could not link %s to %s", src, dest)
	}
	return nil
}

// UnlinkContents will unlink the contents of src to dest if the links exist
// WARNING: on windows smartlinks are hard links, and relating hard links back to their source is non-trivial, so instead
// we just delete the target path. If the user modified the target in any way their changes will be lost.
func UnlinkContents(src, dest string) error {
	if !fileutils.DirExists(dest) {
		return errs.New("dest dir does not exist: %s", dest)
	}

	var err error
	src, dest, err = resolvePaths(src, dest)
	if err != nil {
		return errs.Wrap(err, "Could not resolve src and dest paths")
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return errs.Wrap(err, "Reading dir %s failed", dest)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())
		if !fileutils.TargetExists(destPath) {
			logging.Warning("Could not unlink '%s' as it does not exist, it may have already been removed by another artifact", destPath)
			continue
		}

		if fileutils.IsDir(destPath) {
			if err := UnlinkContents(srcPath, destPath); err != nil {
				return err // Not wrapping here cause it'd just repeat the same error due to the recursion
			}
		} else {
			if err := os.Remove(destPath); err != nil {
				return errs.Wrap(err, "Could not delete %s", destPath)
			}
		}
	}

	// Clean up empty dir afterwards
	isEmpty, err := fileutils.IsEmptyDir(dest)
	if err != nil {
		return errs.Wrap(err, "Could not check if dir %s is empty", dest)
	}
	if isEmpty {
		if err := os.Remove(dest); err != nil {
			return errs.Wrap(err, "Could not delete dir %s", dest)
		}
	}

	return nil
}

// resolvePaths will resolve src and dest to absolute paths and return them.
// This is to ensure that we're always comparing apples to apples when doing string comparisons on paths.
func resolvePaths(src, dest string) (string, string, error) {
	var err error
	src, err = fileutils.ResolveUniquePath(src)
	if err != nil {
		return "", "", errs.Wrap(err, "Could not resolve src path")
	}
	dest, err = fileutils.ResolveUniquePath(dest)
	if err != nil {
		return "", "", errs.Wrap(err, "Could not resolve dest path")
	}

	return src, dest, nil
}
