package fileutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

// FailFindInPathNotFound indicates the specified file was not found in the given path or parent directories
var FailFindInPathNotFound = failures.Type("fileutils.fail.notfoundinpath", failures.FailNotFound, failures.FailNonFatal)

// FailMoveSourceNotDirectory indicates the specified source to be moved is not a directory
var FailMoveSourceNotDirectory = failures.Type("fileutils.fail.move.sourcenotdirectory", failures.FailIO)

// FailMoveDestinationNotDirectory indicates the specified source to be moved is not a directory
var FailMoveDestinationNotDirectory = failures.Type("fileutils.fail.move.destinationnotdirectory", failures.FailIO)

// FailMoveDestinationExists indicates the specified destination to move to already exists
var FailMoveDestinationExists = failures.Type("fileutils.fail.movedestinationexists", failures.FailIO)

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

	findBytes := []byte(find)
	replaceBytes := []byte(replace)
	replaceBytesLen := len(replaceBytes)

	// Check if the file is a binary file. If so, the search and replace byte
	// arrays must be of equal length (replacement being NUL-padded as necessary).
	var replaceRegex *regexp.Regexp
	quoteEscapeFind := regexp.QuoteMeta(find)
	if IsBinary(fileBytes) {
		logging.Debug("Assuming file '%s' is a binary file", filename)

		regexExpandBytes := []byte("${1}")
		// Must account for the expand characters (ie. '${1}') in the
		// replacement bytes in order for the binary paddding to be correct
		replaceBytes = append(replaceBytes, regexExpandBytes...)

		// Replacement regex for binary files must account for null characters
		replaceRegex = regexp.MustCompile(fmt.Sprintf(`%s([^\x00]*)`, quoteEscapeFind))
		if replaceBytesLen > len(findBytes) {
			logging.Errorf("Replacement text too long: %s, original text: %s", string(replaceBytes), string(findBytes))
			return errors.New("replacement text cannot be longer than search text in a binary file")
		} else if len(findBytes) > replaceBytesLen {
			// Pad replacement with NUL bytes.
			logging.Debug("Padding replacement text by %d byte(s)", len(findBytes)-len(replaceBytes))
			paddedReplaceBytes := make([]byte, len(findBytes)+len(regexExpandBytes))
			copy(paddedReplaceBytes, replaceBytes)
			replaceBytes = paddedReplaceBytes
		}
	} else {
		replaceRegex = regexp.MustCompile(fmt.Sprintf(`%s`, quoteEscapeFind))
		logging.Debug("Assuming file '%s' is a text file", filename)
	}

	replaced := replaceRegex.ReplaceAll(fileBytes, replaceBytes)
	buffer := bytes.NewBuffer([]byte{})
	buffer.Write(replaced)

	return WriteFile(filename, buffer.Bytes()).ToError()
}

// ReplaceAllInDirectory walks the given directory and invokes ReplaceAll on each file
func ReplaceAllInDirectory(path, find string, replace string, include includeFunc) error {
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		return ReplaceAll(path, find, replace, include)
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
func Hash(path string) (string, *failures.Failure) {
	hasher := sha256.New()
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", failures.FailIO.New(fmt.Sprintf("Cannot read file: %s, %s", path, err))
	}
	hasher.Write(b)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HashDirectory will sha256 hash the given directory
func HashDirectory(path string) (string, *failures.Failure) {
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
		return "", failures.FailIO.New(fmt.Sprintf("Cannot hash directory: %s, %s", path, err))
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// Mkdir is a small helper function to create a directory if it doesnt already exist
func Mkdir(path string, subpath ...string) *failures.Failure {
	if len(subpath) > 0 {
		subpathStr := filepath.Join(subpath...)
		path = filepath.Join(path, subpathStr)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, DirMode)
		if err != nil {
			return failures.FailIO.Wrap(err, fmt.Sprintf("Path: %s", path))
		}
	}
	return nil
}

// MkdirUnlessExists will make the directory structure if it doesn't already exists
func MkdirUnlessExists(path string) *failures.Failure {
	if DirExists(path) {
		return nil
	}
	return Mkdir(path)
}

// CopyFile copies a file from one location to another
func CopyFile(src, target string) *failures.Failure {
	in, err := os.Open(src)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	defer in.Close()

	// Create target directory if it doesn't exist
	dir := filepath.Dir(target)
	fail := MkdirUnlessExists(dir)
	if fail != nil {
		return fail
	}

	// Create target file
	out, err := os.Create(target)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	defer out.Close()

	// Copy bytes to target file
	_, err = io.Copy(out, in)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	err = out.Close()
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	return nil
}

// ReadFileUnsafe is an unsafe version of ioutil.ReadFile, DO NOT USE THIS OUTSIDE OF TESTS
func ReadFileUnsafe(src string) []byte {
	b, err := ioutil.ReadFile(src)
	if err != nil {
		log.Fatalf("Cannot read file: %s, error: %s", src, err.Error())
	}
	return b
}

// ReadFile reads the content of a file
func ReadFile(filePath string) ([]byte, *failures.Failure) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}
	return b, nil
}

// WriteFile writes data to a file, if it exists it is overwritten, if it doesn't exist it is created and data is written
func WriteFile(filePath string, data []byte) *failures.Failure {
	fail := MkdirUnlessExists(filepath.Dir(filePath))
	if fail != nil {
		return fail
	}

	// make the target file temporarily writable
	fileExists := FileExists(filePath)
	if fileExists {
		stat, _ := os.Stat(filePath)
		if err := os.Chmod(filePath, FileMode); err != nil {
			return failures.FailIO.Wrap(err)
		}
		defer os.Chmod(filePath, stat.Mode().Perm())
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, FileMode)
	if err != nil {
		if !fileExists {
			target := filepath.Dir(filePath)
			err = fmt.Errorf("access to target %q is denied", target)
		}
		return failures.FailIO.Wrap(err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	return nil
}

// AppendToFile appends the data to the file (if it exists) with the given data, if the file doesn't exist
// it is created and the data is written
func AppendToFile(filepath string, data []byte) *failures.Failure {
	return AmendFile(filepath, data, AmendByAppend)
}

// PrependToFile prepends the data to the file (if it exists) with the given data, if the file doesn't exist
// it is created and the data is written
func PrependToFile(filepath string, data []byte) *failures.Failure {
	return AmendFile(filepath, data, AmendByPrepend)
}

// AmendFile amends data to a file, supports append, or prepend
func AmendFile(filePath string, data []byte, flag AmendOptions) *failures.Failure {
	switch flag {
	case
		AmendByAppend, AmendByPrepend:

	default:
		return failures.FailInput.New(locale.Tr("fileutils_err_ammend_file"), filePath)
	}

	fail := Touch(filePath)
	if fail != nil {
		return fail
	}

	b, fail := ReadFile(filePath)
	if fail != nil {
		return fail
	}

	if flag == AmendByPrepend {
		data = append(data, b...)
	} else if flag == AmendByAppend {
		data = append(b, data...)
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY, FileMode)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	return nil
}

// FindFileInPath will find a file by the given file-name in the directory provided or in
// one of the parent directories of that path by walking up the tree. If the file is found,
// the path to that file is returned, otherwise an failure is returned.
func FindFileInPath(dir, filename string) (string, *failures.Failure) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", failures.FailOS.Wrap(err)
	} else if filepath := walkPathAndFindFile(absDir, filename); filepath != "" {
		return filepath, nil
	}
	return "", FailFindInPathNotFound.New("err_file_not_found_in_path", filename, absDir)
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
func Touch(path string) *failures.Failure {
	fail := MkdirUnlessExists(filepath.Dir(path))
	if fail != nil {
		return fail
	}
	file, err := os.OpenFile(path, os.O_CREATE, FileMode)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	if err := file.Close(); err != nil {
		return failures.FailIO.Wrap(err)
	}
	return nil
}

// IsEmptyDir returns true if the directory at the provided path has no files (including dirs) within it.
func IsEmptyDir(path string) (bool, *failures.Failure) {
	dir, err := os.Open(path)
	if err != nil {
		return false, failures.FailIO.Wrap(err)
	}

	files, err := dir.Readdir(1)
	dir.Close()
	if err != nil && err != io.EOF {
		return false, failures.FailIO.Wrap(err)
	}

	return (len(files) == 0), nil
}

// MoveAllFilesCallback is invoked for every file that we move
type MoveAllFilesCallback func()

// MoveAllFilesRecursively moves files and directories from one directory to another.
// Unlike in MoveAllFiles, the destination directory does not need to be empty, and
// may include directories that are moved from the source directory.
// It also counts the moved files for use in a progress bar.
// Warnings are printed if
// - a source file overwrites an existing destination file
// - a sub-directory exists in both the source and and the destination and their permissions do not match
func MoveAllFilesRecursively(fromPath, toPath string, cb MoveAllFilesCallback) *failures.Failure {
	if !DirExists(fromPath) {
		return FailMoveSourceNotDirectory.New("err_os_not_a_directory", fromPath)
	} else if !DirExists(toPath) {
		return FailMoveDestinationNotDirectory.New("err_os_not_a_directory", toPath)
	}

	// read all child files and dirs
	dir, err := os.Open(fromPath)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}
	fileInfos, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	// any found files and dirs
	for _, fileInfo := range fileInfos {
		subFromPath := filepath.Join(fromPath, fileInfo.Name())
		subToPath := filepath.Join(toPath, fileInfo.Name())
		toInfo, err := os.Stat(subToPath)
		// if stat returns, the destination path exists (either file or directory)
		toPathExists := err == nil
		// handle case where destination exists
		if toPathExists {
			if fileInfo.IsDir() != toInfo.IsDir() {
				return FailMoveDestinationExists.New("err_incompatible_move_file_dir", subFromPath, subToPath)
			}
			if fileInfo.Mode() != toInfo.Mode() {
				logging.Warning(locale.T("warn_move_incompatible_modes", subFromPath, subToPath))
			}
		}
		if toPathExists && toInfo.IsDir() {
			fail := MoveAllFilesRecursively(subFromPath, subToPath, cb)
			if fail != nil {
				return fail
			}
			// source path should be empty now
			err := os.Remove(subFromPath)
			if err != nil {
				return failures.FailOS.Wrap(err)
			}
		} else {
			logging.Warning(locale.T("warn_move_destination_overwritten", subFromPath))
			err = os.Rename(subFromPath, subToPath)
			if err != nil {
				return failures.FailOS.Wrap(err)
			}
			cb()
		}
	}
	return nil
}

// MoveAllFiles will move all of the files/dirs within one directory to another directory. Both directories
// must already exist.
func MoveAllFiles(fromPath, toPath string) *failures.Failure {
	if !DirExists(fromPath) {
		return FailMoveSourceNotDirectory.New("err_os_not_a_directory", fromPath)
	} else if !DirExists(toPath) {
		return FailMoveDestinationNotDirectory.New("err_os_not_a_directory", toPath)
	}

	// read all child files and dirs
	dir, err := os.Open(fromPath)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}
	fileInfos, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	// any found files and dirs
	for _, fileInfo := range fileInfos {
		fromPath := filepath.Join(fromPath, fileInfo.Name())
		toPath := filepath.Join(toPath, fileInfo.Name())
		err := os.Rename(fromPath, toPath)
		if err != nil {
			return failures.FailOS.Wrap(err)
		}
	}
	return nil
}

// WriteTempFile writes data to a temp file.
func WriteTempFile(dir, pattern string, data []byte, perm os.FileMode) (string, *failures.Failure) {
	f, err := ioutil.TempFile(dir, pattern)
	if err != nil {
		return "", failures.FailOS.Wrap(err)
	}

	if _, err = f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", failures.FailOS.Wrap(err)
	}

	if err = f.Close(); err != nil {
		os.Remove(f.Name())
		return "", failures.FailOS.Wrap(err)
	}

	if err := os.Chmod(f.Name(), perm); err != nil {
		os.Remove(f.Name())
		return "", failures.FailOS.Wrap(err)
	}

	return f.Name(), nil
}

// CopyFiles will copy all of the files/dirs within one directory to another.
// Both directories must already exist
func CopyFiles(src, dst string) *failures.Failure {
	return copyFiles(src, dst, false)
}

func copyFiles(src, dest string, remove bool) *failures.Failure {
	if !DirExists(src) {
		return failures.FailOS.New("err_os_not_a_directory", src)
	}
	if !DirExists(dest) {
		return failures.FailOS.New("err_os_not_a_directory", dest)
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Lstat(srcPath)
		if err != nil {
			return failures.FailOS.Wrap(err)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			fail := MkdirUnlessExists(destPath)
			if fail != nil {
				return fail
			}
			fail = CopyFiles(srcPath, destPath)
			if fail != nil {
				return fail
			}
		case os.ModeSymlink:
			fail := CopySymlink(srcPath, destPath)
			if fail != nil {
				return fail
			}
		default:
			fail := CopyFile(srcPath, destPath)
			if fail != nil {
				return fail
			}
		}
	}

	if remove {
		if err := os.RemoveAll(src); err != nil {
			return failures.FailOS.Wrap(err)
		}
	}

	return nil
}

// CopySymlink reads the symlink at src and creates a new
// link at dest
func CopySymlink(src, dest string) *failures.Failure {
	link, err := os.Readlink(src)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	err = os.Symlink(link, dest)
	if err != nil {
		return failures.FailOS.Wrap(err)
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

func trialRename(src, dst string) bool {
	if !DirExists(src) {
		return false
	}
	if !DirExists(dst) {
		return false
	}

	tmpFileBase := "test.ext"
	tmpFileData := []byte("data")
	tmpSrcName := filepath.Join(src, tmpFileBase)

	if err := ioutil.WriteFile(tmpSrcName, tmpFileData, 0660); err != nil {
		return false
	}

	cleanupFile := tmpSrcName
	defer func() { _ = os.Remove(cleanupFile) }()

	tmpDstFile := filepath.Join(dst, tmpFileBase)
	if err := os.Rename(tmpSrcName, tmpDstFile); err != nil {
		return false
	}
	cleanupFile = tmpDstFile

	return true
}

// MoveAllFilesCrossDisk will move all of the files/dirs within one directory
// to another directory even across disks. Both directories must already exist.
func MoveAllFilesCrossDisk(src, dst string) *failures.Failure {
	if trialRename(src, dst) {
		return MoveAllFiles(src, dst)
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
	fail := MkdirUnlessExists(path)
	if fail != nil {
		return "", fail
	}

	var err error
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

// IsWritable returns true if the given path is writable
func IsWritable(path string) bool {
	fpath := filepath.Join(path, uuid.New().String())
	if fail := Touch(fpath); fail != nil {
		logging.Debug("Could not create file: %v", fail.ToError())
		return false
	}

	if errr := os.Remove(fpath); errr != nil {
		logging.Debug("Could not clean up test file: %v", errr)
		return false
	}

	return true
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
