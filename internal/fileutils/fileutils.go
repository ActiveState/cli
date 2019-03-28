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
	"path"
	"path/filepath"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
)

// nullByte represents the null-terminator byte
const nullByte byte = 0

// ReplaceAll replaces all instances of search text with replacement text in a
// file, which may be a binary file.
func ReplaceAll(filename, find, replace string) error {
	// Read the file's bytes and create find and replace byte arrays for search
	// and replace.
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	findBytes := []byte(find)
	replaceBytes := []byte(replace)

	// Check if the file is a binary file. If so, the search and replace byte
	// arrays must be of equal length (replacement being NUL-padded as necessary).
	if bytes.IndexByte(fileBytes, nullByte) != -1 {
		logging.Debug("Assuming file '%s' is a binary file", filename)
		if len(replaceBytes) > len(findBytes) {
			return errors.New("replacement text cannot be longer than search text in a binary file")
		} else if len(findBytes) > len(replaceBytes) {
			// Pad replacement with NUL bytes.
			logging.Debug("Padding replacement text by %d byte(s)", len(findBytes)-len(replaceBytes))
			paddedReplaceBytes := make([]byte, len(findBytes))
			copy(paddedReplaceBytes, replaceBytes)
			replaceBytes = paddedReplaceBytes
		}
	}

	chunks := bytes.Split(fileBytes, findBytes)
	if len(chunks) < 2 {
		// nothing to replace
		return nil
	}

	// Open a temporary file for the replacement file and then perform the search
	// and replace.
	tmpfile, err := ioutil.TempFile("", "activestatecli-fileutils")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	for i, chunk := range chunks {
		// Write chunk up to found bytes.
		if _, err := tmpfile.Write(chunk); err != nil {
			tmpfile.Close()
			os.Remove(tmpfile.Name())
			return err
		}
		if i < len(chunks)-1 {
			// Write replacement bytes.
			if _, err := tmpfile.Write(replaceBytes); err != nil {
				tmpfile.Close()
				os.Remove(tmpfile.Name())
				return err
			}
		}
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}

	// make the target file temporarily writable
	stat, _ := os.Stat(filename)
	if err := os.Chmod(filename, os.ModePerm); err != nil {
		return err
	}
	defer func() {
		// put original permissions back on original file
		os.Chmod(filename, stat.Mode().Perm())
	}()

	// we copy file contents instead of renaming the temp file in the event the two files
	// are on different partitions. golang doesn't like to move files across partitions
	// it would seem.
	if failure := CopyFile(tmpfile.Name(), filename); failure != nil {
		return failure.ToError()
	}
	return nil
}

// ReplaceAllInDirectory walks the given directory and invokes ReplaceAll on each file
func ReplaceAllInDirectory(path string, find, replace string) error {
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		return ReplaceAll(path, find, replace)
	})

	if err != nil {
		return err
	}

	return nil
}

// IsExecutable determines if the file at the given path has any execute permissions.
// This function does not care whether the current user can has enough privilege to
// execute the file.
func IsExecutable(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && (stat.Mode()&(0111) > 0)
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

// PathExists checks if the given path exists, this can be a file or a folder
func PathExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
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
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return failures.FailIO.Wrap(err)
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

// WriteFile data to a file, supports overwrite, append, or prepend
// flags:
//   append: 0,
//   overwrite: 1,
//   prepend: 2,
func WriteFile(filepath string, content string, flag int) *failures.Failure {
	switch flag {
	case
		0, 1, 2:

	default:
		fail := failures.FailInput.New(fmt.Sprintf("Unknown flag for fileutils.WriteFile: %d", flag))
		return fail
	}

	data := []byte(content)
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	if flag == 2 {
		data = append(data, b...)
	} else if flag == 0 {
		data = append(b, data...)
	}

	f, err := os.OpenFile(filepath, os.O_WRONLY, 0600)
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
	return "", failures.FailNotFound.New("err_file_not_found_in_path", filename, absDir)
}

// walkPathAndFindFile finds a file in the provided directory or one of its parent directories.
// walkPathAndFindFile prefers an absolute directory path.
func walkPathAndFindFile(dir, filename string) string {
	if file := path.Join(dir, filename); FileExists(file) {
		return file
	} else if parentDir := path.Dir(dir); parentDir != dir {
		return walkPathAndFindFile(parentDir, filename)
	}
	return ""
}

// Touch will attempt to "touch" a given filename by trying to open it read-only or create
// the file with 0644 perms if it does not exist. A File handle will be returned if no issues
// arise. You will need to Close() the file.
func Touch(filepath string) (*os.File, *failures.Failure) {
	file, err := os.OpenFile(filepath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}
	return file, nil
}

// IsEmptyDir returns true if the directory at the provided path has no files (including dirs) within it.
func IsEmptyDir(path string) (bool, *failures.Failure) {
	dir, err := os.Open(path)
	if err != nil {
		return false, failures.FailIO.Wrap(err)
	}

	files, err := dir.Readdir(1)
	if err != nil && err != io.EOF {
		return false, failures.FailIO.Wrap(err)
	}

	return (len(files) == 0), nil
}

// MoveAllFiles will move all of the files/dirs within one directory to another directory. Both directories
// must already exist.
func MoveAllFiles(fromPath, toPath string) *failures.Failure {
	if !DirExists(fromPath) {
		return failures.FailOS.New("err_os_not_a_directory", fromPath)
	} else if !DirExists(toPath) {
		return failures.FailOS.New("err_os_not_a_directory", toPath)
	}

	// read all child files and dirs
	dir, err := os.Open(fromPath)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	// any found files and dirs
	for _, fileInfo := range fileInfos {
		err := os.Rename(path.Join(fromPath, fileInfo.Name()), path.Join(toPath, fileInfo.Name()))
		if err != nil {
			return failures.FailOS.Wrap(err)
		}
	}
	return nil
}
