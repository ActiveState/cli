package fileutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/iafan/cwalk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

// nullByte represents the null-terminator byte
const nullByte byte = 0

// FileMode is the mode used for created files
const FileMode = 0644

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

var (
	ErrorFileNotFound = errs.New("File could not be found")
)

type includeFunc func(path string, contents []byte) (include bool)

// ReplaceAll replaces all instances of search text with replacement text in a
// file, which may be a binary file.
func ReplaceAll(filename, find string, replace string, include includeFunc) error {
	// Read the file's bytes and create find and replace byte arrays for search
	// and replace.
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if !include(filename, fileBytes) {
		return nil
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
		//logging.Debug("Assuming file '%s' is a binary file", filename)

		regexExpandBytes := []byte("${1}")
		// Must account for the expand characters (ie. '${1}') in the
		// replacement bytes in order for the binary paddding to be correct
		replaceBytes = append(replaceBytes, regexExpandBytes...)

		// Replacement regex for binary files must account for null characters
		replaceRegex = regexp.MustCompile(fmt.Sprintf(`%s([^\x00]*)`, quoteEscapeFind))
		if replaceBytesLen > len(findBytes) {
			logging.Errorf("Replacement text too long: %s, original text: %s", string(replaceBytes), string(findBytes))
			return false, nil, errors.New("replacement text cannot be longer than search text in a binary file")
		} else if len(findBytes) > replaceBytesLen {
			// Pad replacement with NUL bytes.
			//logging.Debug("Padding replacement text by %d byte(s)", len(findBytes)-len(replaceBytes))
			paddedReplaceBytes := make([]byte, len(findBytes)+len(regexExpandBytes))
			copy(paddedReplaceBytes, replaceBytes)
			replaceBytes = paddedReplaceBytes
		}
	} else {
		replaceRegex = regexp.MustCompile(fmt.Sprintf(`%s`, quoteEscapeFind))
		//logging.Debug("Assuming file '%s' is a text file", filename)
	}

	replaced := replaceRegex.ReplaceAll(buf, replaceBytes)

	return !bytes.Equal(replaced, buf), replaced, nil
}

// ReplaceAllInDirectory walks the given directory and invokes ReplaceAll on each file
func ReplaceAllInDirectory(path, find string, replace string, include includeFunc) error {
	err := cwalk.Walk(path, func(subpath string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		return ReplaceAll(filepath.Join(path, subpath), find, replace, include)
	})

	if err != nil {
		return err
	}

	return nil
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
	_, err1 := os.Stat(path)
	_, err2 := os.Readlink(path) // os.Stat returns false on Symlinks that don't point to a valid file
	return err1 == nil || err2 == nil
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

// Hash will sha256 hash the given file
func Hash(path string) (string, error) {
	hasher := sha256.New()
	b, err := ioutil.ReadFile(path)
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

		b, err := ioutil.ReadFile(path)
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
	return nil
}

// ReadFileUnsafe is an unsafe version of ioutil.ReadFile, DO NOT USE THIS OUTSIDE OF TESTS
func ReadFileUnsafe(src string) []byte {
	b, err := ioutil.ReadFile(src)
	if err != nil {
		panic(fmt.Sprintf("Cannot read file: %s, error: %s", src, err.Error()))
	}
	return b
}

// ReadFile reads the content of a file
func ReadFile(filePath string) ([]byte, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errs.Wrap(err, "ioutil.ReadFile %s failed", filePath)
	}
	return b, nil
}

// WriteFile writes data to a file, if it exists it is overwritten, if it doesn't exist it is created and data is written
func WriteFile(filePath string, data []byte) error {
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
		defer os.Chmod(filePath, stat.Mode().Perm())
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, FileMode)
	if err != nil {
		if !fileExists {
			target := filepath.Dir(filePath)
			err = fmt.Errorf("access to target %q is denied", target)
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
	case
		AmendByAppend, AmendByPrepend:

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
		toPathExists := err == nil
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
				logging.Error("Failed to remove destination file %s: %v", subToPath, err)
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
			return errs.Wrap(err, "os.Rename %s:%s failed", subFromPath, subToPath)
		}
		cb(subFromPath, subToPath)
	}
	return nil
}

// CopyAndRenameFiles copies files from fromDir to toDir.
// If the target file exists already, it is first copied next to it, and then overwritten by renaming it.
// This method is more robust and than copying directly, in case the target file is opened or executed.
func CopyAndRenameFiles(fromPath, toPath string) error {
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
		if TargetExists(toPath) {
			tmpToPath := fmt.Sprintf("%s.new", toPath)
			err := CopyFile(fromPath, tmpToPath)
			if err != nil {
				return errs.Wrap(err, "failed to copy %s -> %s", fromPath, tmpToPath)
			}
			err = os.Chmod(tmpToPath, fileInfo.Mode())
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
			err = os.Chmod(toPath, fileInfo.Mode())
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
func WriteTempFile(dir, pattern string, data []byte, perm os.FileMode) (string, error) {
	f, err := ioutil.TempFile(dir, pattern)
	if err != nil {
		return "", errs.Wrap(err, "ioutil.TempFile %s (%s) failed", dir, pattern)
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

// CopyFiles will copy all of the files/dirs within one directory to another.
// Both directories must already exist
func CopyFiles(src, dst string) error {
	return copyFiles(src, dst, false)
}

func copyFiles(src, dest string, remove bool) error {
	if !DirExists(src) {
		return locale.NewError("err_os_not_a_directory", "", src)
	}
	if !DirExists(dest) {
		return locale.NewError("err_os_not_a_directory", "", dest)
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return errs.Wrap(err, "ioutil.ReadDir %s failed", src)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Lstat(srcPath)
		if err != nil {
			return errs.Wrap(err, "os.Lstat %s failed", srcPath)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			err := MkdirUnlessExists(destPath)
			if err != nil {
				return errs.Wrap(err, "MkdirUnlessExists %s failed", destPath)
			}
			err = CopyFiles(srcPath, destPath)
			if err != nil {
				return errs.Wrap(err, "CopyFiles %s:%s failed", srcPath, destPath)
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
func TempFileUnsafe() *os.File {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		panic(fmt.Sprintf("Could not create tempFile: %v", err))
	}
	return f
}

// TempDirUnsafe returns a temp path or panics if it cannot be created
// This is for use in tests, do not use it outside tests!
func TempDirUnsafe() string {
	f, err := ioutil.TempDir("", "")
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
		logging.Error("Move all files failed with error: %s. Falling back to copy files", err)
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
			logging.Error("Error walking filepath at: %s", path)
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

// HomeDir returns the users homedir
func HomeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return usr.HomeDir, nil
}

// IsDir returns true if the given path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		logging.Debug("Could not stat path: %s, got error: %v", path, err)
		return false
	}
	return info.IsDir()
}

// ResolvePath gets the absolute location of the provided path and
// fully evaluates the result if it is a symlink.
func ResolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errs.Wrap(err, "cannot get absolute filepath of %q", path)
	}

	if !TargetExists(path) {
		return absPath, nil
	}

	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", errs.Wrap(err, "cannot evaluate symlink %q", absPath)
	}

	return evalPath, nil
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

// ListDir recursively lists filepaths under the given sourcePath
// This does not follow symlinks
func ListDir(sourcePath string, includeDirs bool) []string {
	result := []string{}
	filepath.Walk(sourcePath, func(path string, f os.FileInfo, err error) error {
		if includeDirs == false && f.IsDir() {
			return nil
		}
		result = append(result, path)
		return nil
	})
	return result
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
