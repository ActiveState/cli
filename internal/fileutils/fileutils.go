package fileutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/gofrs/flock"
	"github.com/labstack/gommon/random"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

// nullByte represents the null-terminator byte
const nullByte byte = 0

// FileMode is the mode used for created files
const FileMode = 0o644

// DirMode is the mode used for created dirs
const DirMode = os.ModePerm

// AmendOptions used to specify write actions for WriteFile
type AmendOptions uint8

const (
	// AmendByAppend content to end of file
	AmendByAppend AmendOptions = iota
	// WriteOverwrite file with contents
	WriteOverwrite
	// AmendByPrepend - add content start of file
	AmendByPrepend
)

var ErrorFileNotFound = errs.New("File could not be found")

// ReplaceAll replaces all instances of search text with replacement text in a
// file, which may be a binary file.
func ReplaceAll(filename, find, replace string) error {
	// Read the file's bytes and create find and replace byte arrays for search
	// and replace.
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	changed, byts, err := replaceInFile(fileBytes, find, replace)
	if err != nil {
		return err
	}

	// skip writing file, if we did not change anything
	if !changed {
		return nil
	}

	if err := WriteFile(filename, byts); err != nil {
		return errs.Wrap(err, "WriteFile %s failed", filename)
	}

	return nil
}

// replaceInFile replaces all occurrences of oldpath with newpath
// For binary files with nul-terminated strings, it ensures that the replaces strings are still valid nul-terminated strings and the returned buffer has the same size as the input buffer buf.
// The first return argument denotes whether at least one file has been replaced
func replaceInFile(buf []byte, oldpath, newpath string) (bool, []byte, error) {
	findBytes := []byte(oldpath)
	replaceBytes := []byte(newpath)
	replaceBytesLen := len(replaceBytes)

	// Check if the file is a binary file. If so, the search and replace byte
	// arrays must be of equal length (replacement being NUL-padded as necessary).
	var replaceRegex *regexp.Regexp
	quoteEscapeFind := regexp.QuoteMeta(oldpath)

	// Ensure we replace both types of backslashes on Windows
	if runtime.GOOS == "windows" {
		quoteEscapeFind = strings.ReplaceAll(quoteEscapeFind, `\\`, `(\\|\\\\)`)
	}
	if IsBinary(buf) {
		// logging.Debug("Assuming file '%s' is a binary file", filename)

		regexExpandBytes := []byte("${1}")
		// Must account for the expand characters (ie. '${1}') in the
		// replacement bytes in order for the binary paddding to be correct
		replaceBytes = append(replaceBytes, regexExpandBytes...)

		// Replacement regex for binary files must account for null characters
		replaceRegex = regexp.MustCompile(fmt.Sprintf(`%s([^\x00]*)`, quoteEscapeFind))
		if replaceBytesLen > len(findBytes) {
			multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Replacement text too long: %s, original text: %s", string(replaceBytes), string(findBytes))
			return false, nil, errors.New("replacement text cannot be longer than search text in a binary file")
		} else if len(findBytes) > replaceBytesLen {
			// Pad replacement with NUL bytes.
			// logging.Debug("Padding replacement text by %d byte(s)", len(findBytes)-len(replaceBytes))
			paddedReplaceBytes := make([]byte, len(findBytes)+len(regexExpandBytes))
			copy(paddedReplaceBytes, replaceBytes)
			replaceBytes = paddedReplaceBytes
		}
	} else {
		replaceRegex = regexp.MustCompile(quoteEscapeFind)
		// logging.Debug("Assuming file '%s' is a text file", filename)
	}

	replaced := replaceRegex.ReplaceAll(buf, replaceBytes)

	return !bytes.Equal(replaced, buf), replaced, nil
}

// IsSymlink checks if a path is a symlink
func IsSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeSymlink) == os.ModeSymlink
}

// IsBinary checks if the given bytes are for a binary file
func IsBinary(fileBytes []byte) bool {
	return bytes.IndexByte(fileBytes, nullByte) != -1
}

// TargetExists checks if the given file or folder exists
func TargetExists(path string) bool {
	if FileExists(path) || DirExists(path) {
		return true
	}

	_, err1 := os.Stat(path)
	_, err2 := os.Readlink(path) // os.Stat returns false on Symlinks that don't point to a valid file
	_, err3 := os.Lstat(path)    // for links where os.Stat and os.Readlink fail (e.g. Windows socket files)
	return err1 == nil || err2 == nil || err3 == nil
}

// FileExists checks if the given file (not folder) exists
func FileExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	mode := fi.Mode()
	return mode.IsRegular()
}

// DirExists checks if the given directory exists
func DirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	mode := fi.Mode()
	return mode.IsDir()
}

// Sha256Hash will sha256 hash the given file
func Sha256Hash(path string) (string, error) {
	hasher := sha256.New()
	b, err := os.ReadFile(path)
	if err != nil {
		return "", errs.Wrap(err, fmt.Sprintf("Cannot read file: %s", path))
	}
	hasher.Write(b)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HashDirectory will sha256 hash the given directory
func HashDirectory(path string) (string, error) {
	hasher := sha256.New()
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hasher.Write(b)

		return nil
	})
	if err != nil {
		return "", errs.Wrap(err, fmt.Sprintf("Cannot hash directory: %s", path))
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// Mkdir is a small helper function to create a directory if it doesnt already exist
func Mkdir(path string, subpath ...string) error {
	if len(subpath) > 0 {
		subpathStr := filepath.Join(subpath...)
		path = filepath.Join(path, subpathStr)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, DirMode)
		if err != nil {
			return errs.Wrap(err, fmt.Sprintf("MkdirAll failed for path: %s", path))
		}
	}
	return nil
}

// MkdirUnlessExists will make the directory structure if it doesn't already exists
func MkdirUnlessExists(path string) error {
	if DirExists(path) {
		return nil
	}
	return Mkdir(path)
}

// CopyFile copies a file from one location to another
func CopyFile(src, target string) error {
	in, err := os.Open(src)
	if err != nil {
		return errs.Wrap(err, "os.Open %s failed", src)
	}
	defer in.Close()

	inInfo, err := in.Stat()
	if err != nil {
		return errs.Wrap(err, "get file info failed")
	}

	// Create target directory if it doesn't exist
	dir := filepath.Dir(target)
	err = MkdirUnlessExists(dir)
	if err != nil {
		return err
	}

	// Create target file
	out, err := os.Create(target)
	if err != nil {
		return errs.Wrap(err, "os.Create %s failed", target)
	}
	defer out.Close()

	// Copy bytes to target file
	_, err = io.Copy(out, in)
	if err != nil {
		return errs.Wrap(err, "io.Copy failed")
	}
	err = out.Close()
	if err != nil {
		return errs.Wrap(err, "out.Close failed")
	}

	if err := os.Chmod(out.Name(), inInfo.Mode().Perm()); err != nil {
		return errs.Wrap(err, "chmod failed")
	}

	return nil
}

func CopyAsset(assetName, dest string) error {
	asset, err := assets.ReadFileBytes(assetName)
	if err != nil {
		return errs.Wrap(err, "Asset %s failed", assetName)
	}

	err = os.WriteFile(dest, asset, 0o644)
	if err != nil {
		return errs.Wrap(err, "os.WriteFile %s failed", dest)
	}

	return nil
}

func CopyMultipleFiles(files map[string]string) error {
	for src, target := range files {
		err := CopyFile(src, target)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadFileUnsafe is an unsafe version of os.ReadFile, DO NOT USE THIS OUTSIDE OF TESTS
func ReadFileUnsafe(src string) []byte {
	b, err := os.ReadFile(src)
	if err != nil {
		panic(fmt.Sprintf("Cannot read file: %s, error: %s", src, err.Error()))
	}
	return b
}

// ReadFile reads the content of a file
func ReadFile(filePath string) ([]byte, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errs.Wrap(err, "os.ReadFile %s failed", filePath)
	}
	return b, nil
}

// WriteFile writes data to a file, if it exists it is overwritten, if it doesn't exist it is created and data is written
func WriteFile(filePath string, data []byte) (rerr error) {
	err := MkdirUnlessExists(filepath.Dir(filePath))
	if err != nil {
		return err
	}

	// make the target file temporarily writable
	fileExists := FileExists(filePath)
	if fileExists {
		stat, _ := os.Stat(filePath)
		if err := os.Chmod(filePath, FileMode); err != nil {
			return errs.Wrap(err, "os.Chmod %s failed", filePath)
		}
		defer func() {
			err = os.Chmod(filePath, stat.Mode().Perm())
			if err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "os.Chmod %s failed", filePath))
			}
		}()
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, FileMode)
	if err != nil {
		if !fileExists {
			target := filepath.Dir(filePath)
			err = errs.Pack(err, fmt.Errorf("access to target %q is denied", target))
		}
		return errs.Wrap(err, "os.OpenFile %s failed", filePath)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return errs.Wrap(err, "file.Write %s failed", filePath)
	}
	return nil
}

func AmendFileLocked(filePath string, data []byte, flag AmendOptions) error {
	locker := flock.New(filePath + ".lock")

	if err := locker.Lock(); err != nil {
		return errs.Wrap(err, "Could not acquire file lock")
	}

	if err := AmendFile(filePath, data, flag); err != nil {
		return errs.Wrap(err, "Could not write to file")
	}

	return locker.Unlock()
}

// AppendToFile appends the data to the file (if it exists) with the given data, if the file doesn't exist
// it is created and the data is written
func AppendToFile(filepath string, data []byte) error {
	return AmendFile(filepath, data, AmendByAppend)
}

// PrependToFile prepends the data to the file (if it exists) with the given data, if the file doesn't exist
// it is created and the data is written
func PrependToFile(filepath string, data []byte) error {
	return AmendFile(filepath, data, AmendByPrepend)
}

// AmendFile amends data to a file, supports append, or prepend
func AmendFile(filePath string, data []byte, flag AmendOptions) error {
	switch flag {
	case AmendByAppend, AmendByPrepend:

	default:
		return locale.NewInputError("fileutils_err_amend_file", "", filePath)
	}

	err := Touch(filePath)
	if err != nil {
		return errs.Wrap(err, "Touch %s failed", filePath)
	}

	b, err := ReadFile(filePath)
	if err != nil {
		return errs.Wrap(err, "ReadFile %s failed", filePath)
	}

	if flag == AmendByPrepend {
		data = append(data, b...)
	} else if flag == AmendByAppend {
		data = append(b, data...)
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY, FileMode)
	if err != nil {
		return errs.Wrap(err, "os.OpenFile %s failed", filePath)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return errs.Wrap(err, "file.Write %s failed", filePath)
	}
	return nil
}

// FindFileInPath will find a file by the given file-name in the directory provided or in
// one of the parent directories of that path by walking up the tree. If the file is found,
// the path to that file is returned, otherwise an failure is returned.
func FindFileInPath(dir, filename string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", errs.Wrap(err, "filepath.Abs %s failed", dir)
	} else if filepath := walkPathAndFindFile(absDir, filename); filepath != "" {
		return filepath, nil
	}
	return "", locale.WrapError(ErrorFileNotFound, "err_file_not_found_in_path", "", filename, absDir)
}

// walkPathAndFindFile finds a file in the provided directory or one of its parent directories.
// walkPathAndFindFile prefers an absolute directory path.
func walkPathAndFindFile(dir, filename string) string {
	if file := filepath.Join(dir, filename); FileExists(file) {
		return file
	} else if parentDir := filepath.Dir(dir); parentDir != dir {
		return walkPathAndFindFile(parentDir, filename)
	}
	return ""
}

// Touch will attempt to "touch" a given filename by trying to open it read-only or create
// the file with 0644 perms if it does not exist.
func Touch(path string) error {
	err := MkdirUnlessExists(filepath.Dir(path))
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE, FileMode)
	if err != nil {
		return errs.Wrap(err, "os.OpenFile %s failed", path)
	}
	if err := file.Close(); err != nil {
		return errs.Wrap(err, "file.Close %s failed", path)
	}
	return nil
}

// TouchFileUnlessExists will attempt to "touch" a given filename if it doesn't already exists
func TouchFileUnlessExists(path string) error {
	if TargetExists(path) {
		return nil
	}
	return Touch(path)
}

// IsEmptyDir returns true if the directory at the provided path has no files (including dirs) within it.
func IsEmptyDir(path string) (bool, error) {
	dir, err := os.Open(path)
	if err != nil {
		return false, errs.Wrap(err, "os.Open %s failed", path)
	}

	files, err := dir.Readdir(1)
	dir.Close()
	if err != nil && err != io.EOF {
		return false, errs.Wrap(err, "dir.Readdir %s failed", path)
	}

	return (len(files) == 0), nil
}

// MoveAllFilesCallback is invoked for every file that we move
type MoveAllFilesCallback func(fromPath, toPath string)

// MoveAllFilesRecursively moves files and directories from one directory to another.
// Unlike in MoveAllFiles, the destination directory does not need to be empty, and
// may include directories that are moved from the source directory.
// It also counts the moved files for use in a progress bar.
// Warnings are printed if
// - a source file overwrites an existing destination file
// - a sub-directory exists in both the source and and the destination and their permissions do not match
func MoveAllFilesRecursively(fromPath, toPath string, cb MoveAllFilesCallback) error {
	if !DirExists(fromPath) {
		return locale.NewError("err_os_not_a_directory", "", fromPath)
	} else if !DirExists(toPath) {
		return locale.NewError("err_os_not_a_directory", "", toPath)
	}

	// read all child files and dirs
	dir, err := os.Open(fromPath)
	if err != nil {
		return errs.Wrap(err, "os.Open %s failed", fromPath)
	}
	fileInfos, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		return errs.Wrap(err, "dir.Readdir %s failed", fromPath)
	}

	// any found files and dirs
	for _, fileInfo := range fileInfos {
		subFromPath := filepath.Join(fromPath, fileInfo.Name())
		subToPath := filepath.Join(toPath, fileInfo.Name())
		toInfo, err := os.Lstat(subToPath)
		// if stat returns, the destination path exists (either file or directory)
		toPathExists := toInfo != nil && err == nil
		// handle case where destination exists
		if toPathExists {
			if fileInfo.IsDir() != toInfo.IsDir() {
				return locale.NewError("err_incompatible_move_file_dir", "", subFromPath, subToPath)
			}
			if fileInfo.Mode() != toInfo.Mode() {
				logging.Warning(locale.Tr("warn_move_incompatible_modes", "", subFromPath, subToPath))
			}

			if !toInfo.IsDir() {
				// If the subToPath file exists, we remove it first - in order to ensure compatibility between platforms:
				// On Windows, the following renaming step can otherwise fail if subToPath is read-only (file removal is allowed)
				err = os.Remove(subToPath)
				if err != nil {
					multilog.Error("Failed to remove file scheduled to be overwritten: %s (file mode: %#o): %v", subToPath, toInfo.Mode(), err)
				}
			}
		}

		// If we are moving to a directory, call function recursively to overwrite and add files in that directory
		if fileInfo.IsDir() {
			// create target directories that don't exist yet
			if !toPathExists {
				err = Mkdir(subToPath)
				if err != nil {
					return locale.WrapError(err, "err_move_create_directory", "Failed to create directory {{.V0}}", subToPath)
				}
				err = os.Chmod(subToPath, fileInfo.Mode())
				if err != nil {
					return locale.WrapError(err, "err_move_set_dir_permissions", "Failed to set file mode for directory {{.V0}}", subToPath)
				}
			}
			err := MoveAllFilesRecursively(subFromPath, subToPath, cb)
			if err != nil {
				return err
			}
			// source path should be empty now
			err = os.Remove(subFromPath)
			if err != nil {
				return errs.Wrap(err, "os.Remove %s failed", subFromPath)
			}

			cb(subFromPath, subToPath)
			continue
		}

		err = os.Rename(subFromPath, subToPath)
		if err != nil {
			var mode fs.FileMode
			if toPathExists {
				mode = toInfo.Mode()
			}
			return errs.Wrap(err, "os.Rename %s:%s failed (file mode: %#o)", subFromPath, subToPath, mode)
		}
		cb(subFromPath, subToPath)
	}
	return nil
}

// CopyAndRenameFiles copies files from fromDir to toDir.
// If the target file exists already, the source file is first copied next to the target file, and then overwrites the target by renaming the source.
// This method is more robust and than copying directly, in case the target file is opened or executed.
func CopyAndRenameFiles(fromPath, toPath string, exclude ...string) error {
	logging.Debug("Copying files from %s to %s", fromPath, toPath)

	if !DirExists(fromPath) {
		return locale.NewError("err_os_not_a_directory", "", fromPath)
	} else if !DirExists(toPath) {
		return locale.NewError("err_os_not_a_directory", "", toPath)
	}

	// read all child files and dirs
	files, err := ListDir(fromPath, true)
	if err != nil {
		return errs.Wrap(err, "Could not ListDir %s", fromPath)
	}

	// any found files and dirs
	for _, file := range files {
		if funk.Contains(exclude, file.Name()) {
			continue
		}

		rpath := file.RelativePath()
		fromPath := filepath.Join(fromPath, rpath)
		toPath := filepath.Join(toPath, rpath)

		if file.IsDir() {
			if err := MkdirUnlessExists(toPath); err != nil {
				return errs.Wrap(err, "Could not create dir: %s", toPath)
			}
			continue
		}

		finfo, err := file.Info()
		if err != nil {
			return errs.Wrap(err, "Could not get file info for %s", file.RelativePath())
		}

		if TargetExists(toPath) {
			tmpToPath := fmt.Sprintf("%s.new", toPath)
			err := CopyFile(fromPath, tmpToPath)
			if err != nil {
				return errs.Wrap(err, "failed to copy %s -> %s", fromPath, tmpToPath)
			}
			err = os.Chmod(tmpToPath, finfo.Mode())
			if err != nil {
				return errs.Wrap(err, "failed to set file permissions for %s", tmpToPath)
			}
			err = os.Rename(tmpToPath, toPath)
			if err != nil {
				// cleanup
				_ = os.Remove(tmpToPath)
				return errs.Wrap(err, "os.Rename %s -> %s failed", tmpToPath, toPath)
			}
		} else {
			err := CopyFile(fromPath, toPath)
			if err != nil {
				return errs.Wrap(err, "Copy %s -> %s failed", fromPath, toPath)
			}
			err = os.Chmod(toPath, finfo.Mode())
			if err != nil {
				return errs.Wrap(err, "failed to set file permissions for %s", toPath)
			}
		}
	}
	return nil
}

// MoveAllFiles will move all of the files/dirs within one directory to another directory. Both directories
// must already exist.
func MoveAllFiles(fromPath, toPath string) error {
	if !DirExists(fromPath) {
		return locale.NewError("err_os_not_a_directory", "", fromPath)
	} else if !DirExists(toPath) {
		return locale.NewError("err_os_not_a_directory", "", toPath)
	}

	// read all child files and dirs
	dir, err := os.Open(fromPath)
	if err != nil {
		return errs.Wrap(err, "os.Open %s failed", fromPath)
	}
	fileInfos, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		return errs.Wrap(err, "dir.Readdir %s failed", fromPath)
	}

	// any found files and dirs
	for _, fileInfo := range fileInfos {
		fromPath := filepath.Join(fromPath, fileInfo.Name())
		toPath := filepath.Join(toPath, fileInfo.Name())
		err := os.Rename(fromPath, toPath)
		if err != nil {
			return errs.Wrap(err, "os.Rename %s:%s failed", fromPath, toPath)
		}
	}
	return nil
}

// WriteTempFile writes data to a temp file.
func WriteTempFile(pattern string, data []byte) (string, error) {
	tempDir := os.TempDir()
	return WriteTempFileToDir(tempDir, pattern, data, os.ModePerm)
}

// WriteTempFileToDir writes data to a temp file in the given dir
func WriteTempFileToDir(dir, pattern string, data []byte, perm os.FileMode) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", errs.Wrap(err, "os.CreateTemp %s (%s) failed", dir, pattern)
	}

	if _, err = f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", errs.Wrap(err, "f.Write %s failed", f.Name())
	}

	if err = f.Close(); err != nil {
		os.Remove(f.Name())
		return "", errs.Wrap(err, "f.Close %s failed", f.Name())
	}

	if err := os.Chmod(f.Name(), perm); err != nil {
		os.Remove(f.Name())
		return "", errs.Wrap(err, "os.Chmod %s failed", f.Name())
	}

	return f.Name(), nil
}

type DirReader interface {
	ReadDir(string) ([]os.DirEntry, error)
}

func CopyFilesDirReader(reader DirReader, src, dst, placeholderFileName string) error {
	entries, err := reader.ReadDir(src)
	if err != nil {
		return errs.Wrap(err, "reader.ReadDir %s failed", src)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		switch entry.Type() & os.ModeType {
		case os.ModeDir:
			err := MkdirUnlessExists(destPath)
			if err != nil {
				return errs.Wrap(err, "MkdirUnlessExists %s failed", destPath)
			}

			err = CopyFilesDirReader(reader, srcPath, destPath, placeholderFileName)
			if err != nil {
				return errs.Wrap(err, "CopyFiles %s:%s failed", srcPath, destPath)
			}
		case os.ModeSymlink:
			err := CopySymlink(srcPath, destPath)
			if err != nil {
				return errs.Wrap(err, "CopySymlink %s:%s failed", srcPath, destPath)
			}
		default:
			if entry.Name() == placeholderFileName {
				continue
			}

			err := CopyAsset(srcPath, destPath)
			if err != nil {
				return errs.Wrap(err, "CopyFile %s:%s failed", srcPath, destPath)
			}
		}
	}

	return nil
}

// CopyFiles will copy all of the files/dirs within one directory to another.
// Both directories must already exist
func CopyFiles(src, dst string) error {
	return copyFiles(src, dst, false)
}

type ErrAlreadyExist struct {
	Path string
}

func (e *ErrAlreadyExist) Error() string {
	return fmt.Sprintf("file already exists: %s", e.Path)
}

func copyFiles(src, dest string, remove bool) error {
	if !DirExists(src) {
		return locale.NewError("err_os_not_a_directory", "", src)
	}
	if !DirExists(dest) {
		return locale.NewError("err_os_not_a_directory", "", dest)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return errs.Wrap(err, "os.ReadDir %s failed", src)
	}

	var errAlreadyExist error
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Lstat(srcPath)
		if err != nil {
			return errs.Wrap(err, "os.Lstat %s failed", srcPath)
		}

		if !fileInfo.IsDir() && TargetExists(destPath) {
			errAlreadyExist = errs.Pack(errAlreadyExist, &ErrAlreadyExist{destPath})
			continue
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			err := MkdirUnlessExists(destPath)
			if err != nil {
				return errs.Wrap(err, "MkdirUnlessExists %s failed", destPath)
			}
			err = copyFiles(srcPath, destPath, remove)
			if err != nil {
				if errors.As(err, ptr.To(&ErrAlreadyExist{})) {
					errAlreadyExist = errs.Pack(errAlreadyExist, err)
				} else {
					return errs.Wrap(err, "CopyFiles %s:%s failed", srcPath, destPath)
				}
			}
		case os.ModeSymlink:
			err := CopySymlink(srcPath, destPath)
			if err != nil {
				return errs.Wrap(err, "CopySymlink %s:%s failed", srcPath, destPath)
			}
		default:
			err := CopyFile(srcPath, destPath)
			if err != nil {
				return errs.Wrap(err, "CopyFile %s:%s failed", srcPath, destPath)
			}
		}
	}

	if remove {
		if err := os.RemoveAll(src); err != nil {
			return errs.Wrap(err, "os.RemovaAll %s failed", src)
		}
	}

	// If some files already exist we want to error on this, but only after all other remaining files have been copied.
	// If ANY other type of error occurs then we don't bubble this up as this is the only error we handle that's non-critical.
	if errAlreadyExist != nil {
		return errAlreadyExist
	}

	return nil
}

// CopySymlink reads the symlink at src and creates a new
// link at dest
func CopySymlink(src, dest string) error {
	link, err := os.Readlink(src)
	if err != nil {
		return errs.Wrap(err, "os.Readlink %s failed", src)
	}

	err = os.Symlink(link, dest)
	if err != nil {
		return errs.Wrap(err, "os.Symlink %s:%s failed", link, dest)
	}

	return nil
}

// TempFileUnsafe returns a tempfile handler or panics if it cannot be created
// This is for use in tests, do not use it outside tests!
func TempFileUnsafe(dir, pattern string) *os.File {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		panic(fmt.Sprintf("Could not create tempFile: %v", err))
	}
	return f
}

func TempFilePath(dir, pattern string) string {
	if dir == "" {
		dir = os.TempDir()
	}
	fname := random.String(8, random.Alphanumeric)
	if pattern != "" {
		fname = fmt.Sprintf("%s-%s", fname, pattern)
	}
	return filepath.Join(dir, fname)
}

// TempDirUnsafe returns a temp path or panics if it cannot be created
// This is for use in tests, do not use it outside tests!
func TempDirUnsafe() string {
	f, err := os.MkdirTemp("", "")
	if err != nil {
		panic(fmt.Sprintf("Could not create tempDir: %v", err))
	}
	return f
}

func TempDirFromBaseDirUnsafe(baseDir string) string {
	f, err := os.MkdirTemp(baseDir, "")
	if err != nil {
		panic(fmt.Sprintf("Could not create tempDir: %v", err))
	}
	return f
}

// MoveAllFilesCrossDisk will move all of the files/dirs within one directory
// to another directory even across disks. Both directories must already exist.
func MoveAllFilesCrossDisk(src, dst string) error {
	err := MoveAllFiles(src, dst)
	if err != nil {
		multilog.Error("Move all files failed with error: %s. Falling back to copy files", err)
	}

	return copyFiles(src, dst, true)
}

// Join is identical to filepath.Join except that it doesn't clean the input, allowing for
// more consistent behavior
func Join(elem ...string) string {
	for i, e := range elem {
		if e != "" {
			return strings.Join(elem[i:], string(filepath.Separator))
		}
	}
	return ""
}

// PrepareDir prepares a path by ensuring it exists and the path is consistent
func PrepareDir(path string) (string, error) {
	err := MkdirUnlessExists(path)
	if err != nil {
		return "", err
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}

	return path, nil
}

// LogPath will walk the given file path and log the name, permissions, mod
// time, and file size of all files it encounters
func LogPath(path string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			multilog.Error("Error walking filepath at: %s", path)
			return err
		}

		logging.Debug(strings.Join([]string{
			fmt.Sprintf("File name: %s", info.Name()),
			fmt.Sprintf("File permissions: %s", info.Mode()),
			fmt.Sprintf("File mod time: %s", info.ModTime()),
			fmt.Sprintf("File size: %d", info.Size()),
		}, "\n"))
		return nil
	})
}

// IsDir returns true if the given path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ResolvePath gets the absolute location of the provided path and
// fully evaluates the result if it is a symlink.
func ResolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path, errs.Wrap(err, "cannot get absolute filepath of %q", path)
	}

	if !TargetExists(path) {
		return absPath, nil
	}

	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return absPath, errs.Wrap(err, "cannot evaluate symlink %q", absPath)
	}

	return evalPath, nil
}

func ResolvePathIfPossible(path string) string {
	if resolvedPath, err := ResolveUniquePath(path); err == nil {
		return resolvedPath
	}
	return path
}

// PathsEqual checks whether the paths given all resolve to the same path
func PathsEqual(paths ...string) (bool, error) {
	if len(paths) < 2 {
		return false, errs.New("Must supply at least two paths")
	}

	var equalTo string
	for _, path := range paths {
		resolvedPath, err := ResolvePath(path)
		if err != nil {
			return false, errs.Wrap(err, "Could not resolve path: %s", path)
		}
		if equalTo == "" {
			equalTo = resolvedPath
			continue
		}
		if resolvedPath != equalTo {
			return false, nil
		}
	}

	return true, nil
}

// PathContainsParent checks if the directory path is equal to or a child directory
// of the targeted directory. Symlinks are evaluated for this comparison.
func PathContainsParent(path, parentPath string) (bool, error) {
	if path == parentPath {
		return true, nil
	}

	efmt := "cannot resolve %q"

	resPath, err := ResolvePath(path)
	if err != nil {
		return false, errs.Wrap(err, efmt, path)
	}

	resParent, err := ResolvePath(parentPath)
	if err != nil {
		return false, errs.Wrap(err, efmt, parentPath)
	}

	return resolvedPathContainsParent(resPath, resParent), nil
}

func resolvedPathContainsParent(path, parentPath string) bool {
	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	if !strings.HasSuffix(parentPath, string(os.PathSeparator)) {
		parentPath += string(os.PathSeparator)
	}

	return path == parentPath || strings.HasPrefix(path, parentPath)
}

// SymlinkTarget retrieves the target of the given symlink
func SymlinkTarget(symlink string) (string, error) {
	fileInfo, err := os.Lstat(symlink)
	if err != nil {
		return "", errs.Wrap(err, "Could not lstat symlink")
	}

	if fileInfo.Mode()&os.ModeSymlink != os.ModeSymlink {
		return "", errs.New("%s is not a symlink", symlink)
	}

	evalDest, err := os.Readlink(symlink)
	if err != nil {
		return "", errs.Wrap(err, "Could not resolve symlink: %s", symlink)
	}

	return evalDest, nil
}

// ListDirSimple recursively lists filepaths under the given sourcePath
// This does not follow symlinks
func ListDirSimple(sourcePath string, includeDirs bool) ([]string, error) {
	result := []string{}
	err := filepath.WalkDir(sourcePath, func(path string, f fs.DirEntry, err error) error {
		if err != nil {
			return errs.Wrap(err, "Could not walk path: %s", path)
		}
		if !includeDirs && f.IsDir() {
			return nil
		}
		result = append(result, path)
		return nil
	})
	if err != nil {
		return result, errs.Wrap(err, "Could not walk dir: %s", sourcePath)
	}
	return result, nil
}

// ListFilesUnsafe lists filepaths under the given sourcePath non-recursively
func ListFilesUnsafe(sourcePath string) []string {
	result := []string{}
	files, err := os.ReadDir(sourcePath)
	if err != nil {
		panic(fmt.Sprintf("Could not read dir: %s, error: %s", sourcePath, errs.JoinMessage(err)))
	}
	for _, file := range files {
		result = append(result, filepath.Join(sourcePath, file.Name()))
	}
	return result
}

type DirEntry struct {
	fs.DirEntry
	absolutePath string
	rootPath     string
}

func (d DirEntry) AbsolutePath() string {
	return d.absolutePath
}

func (d DirEntry) RelativePath() string {
	// This is a bit awkward, but fs.DirEntry does not give us a relative path to the originally queried dir
	return strings.TrimPrefix(d.absolutePath, d.rootPath)
}

type DirEntries []DirEntry

func (d DirEntries) RelativePaths() []string {
	result := []string{}
	for _, de := range d {
		result = append(result, de.RelativePath())
	}
	return result
}

// ListDir recursively lists filepaths under the given sourcePath
// This does not follow symlinks
func ListDir(sourcePath string, includeDirs bool) (DirEntries, error) {
	result := []DirEntry{}
	sourcePath = filepath.Clean(sourcePath)
	if err := filepath.WalkDir(sourcePath, func(path string, f fs.DirEntry, err error) error {
		if path == sourcePath {
			return nil // I don't know why WalkDir feels the need to include the very dir I queried..
		}
		if err != nil {
			return errs.Wrap(err, "Could not walk path: %s", path)
		}
		if !includeDirs && f.IsDir() {
			return nil
		}
		result = append(result, DirEntry{f, path, sourcePath + string(filepath.Separator)})
		return nil
	}); err != nil {
		return result, errs.Wrap(err, "Could not walk dir: %s", sourcePath)
	}
	return result, nil
}

// PathInList returns whether the provided path list contains the provided
// path.
func PathInList(listSep, pathList, path string) (bool, error) {
	paths := strings.Split(pathList, listSep)
	for _, p := range paths {
		equal, err := PathsEqual(p, path)
		if err != nil {
			return false, err
		}
		if equal {
			return true, nil
		}
	}
	return false, nil
}

func FileContains(path string, searchText []byte) (bool, error) {
	if !TargetExists(path) {
		return false, nil
	}
	b, err := ReadFile(path)
	if err != nil {
		return false, errs.Wrap(err, "Could not read file")
	}
	return bytes.Contains(b, searchText), nil
}

func ModTime(path string) (time.Time, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return time.Now(), errs.Wrap(err, "Could not stat file %s", path)
	}
	return stat.ModTime(), nil
}

func CaseSensitivePath(path string) (string, error) {
	// On Windows Glob may not work with the short path (ie., DOS 8.3 notation)
	path, err := GetLongPathName(path)
	if err != nil {
		return "", errs.Wrap(err, "Failed to get long path name")
	}

	var searchPath string
	if runtime.GOOS != "windows" {
		searchPath = globPath(path)
	} else {
		volume := filepath.VolumeName(path)
		remainder := strings.TrimLeft(path, volume)
		searchPath = filepath.Join(volume, globPath(remainder))
	}

	matches, err := filepath.Glob(searchPath)
	if err != nil {
		return "", errs.Wrap(err, "Failed to search for path")
	}

	if len(matches) == 0 {
		return "", errs.New("Could not find path: %s", path)
	}

	return matches[0], nil
}

// PathsMatch checks if all the given paths resolve to the same value
func PathsMatch(paths ...string) (bool, error) {
	for _, path := range paths[1:] {
		p1, err := ResolvePath(path)
		if err != nil {
			return false, errs.Wrap(err, "Could not resolve path %s", path)
		}
		p2, err := ResolvePath(paths[0])
		if err != nil {
			return false, errs.Wrap(err, "Could not resolve path %s", paths[0])
		}
		if p1 != p2 {
			logging.Debug("Path %s does not match %s", p1, p2)
			return false, nil
		}
	}
	return true, nil
}

func globPath(path string) string {
	var result string
	for _, r := range path {
		if unicode.IsLetter(r) {
			result += fmt.Sprintf("[%c%c]", unicode.ToUpper(r), unicode.ToLower(r))
		} else {
			result += string(r)
		}
	}
	return result
}

// CommonParentPath will return the common parent path of the given paths, provided they share a common path.
// If they do not all share a single common path the result will be empty.
func CommonParentPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	common := paths[0]
	for _, p := range paths[1:] {
		common = commonParentPath(common, p)
		if common == "" {
			return ""
		}
	}

	return common
}

func commonParentPath(a, b string) (result string) {
	isWindowsPath := false
	defer func() {
		if isWindowsPath {
			result = posixPathToWindowsPath(result)
		}
	}()
	common := ""
	ab := windowsPathToPosixPath(a)
	bb := windowsPathToPosixPath(b)
	isWindowsPath = a != ab
	as := strings.Split(ab, "/")
	bs := strings.Split(bb, "/")
	max := min(len(as), len(bs))
	for x := 1; x <= max; x++ {
		ac := strings.Join(as[:x], "/")
		bc := strings.Join(bs[:x], "/")
		if ac != bc {
			return common
		}
		common = ac
	}
	return common
}

func windowsPathToPosixPath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func posixPathToWindowsPath(path string) string {
	return strings.ReplaceAll(path, "/", "\\")
}
