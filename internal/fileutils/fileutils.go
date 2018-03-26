package fileutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
)

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
	if bytes.IndexByte(fileBytes, '0') != -1 {
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

	// Open a temporary file for the replacement file and then perform the search
	// and replace.
	tmpfile, err := ioutil.TempFile("", "activestatecli-fileutils")
	if err != nil {
		return err
	}
	chunks := bytes.Split(fileBytes, findBytes)
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

	// Replace the original file.
	stat, _ := os.Stat(filename)
	if err := os.Chmod(tmpfile.Name(), stat.Mode()); err != nil {
		return err
	}
	return os.Rename(tmpfile.Name(), filename)
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

// Mkdir is a small helper function to create a directory if it doesnt already exist
func Mkdir(parent string, subpath ...string) *failures.Failure {
	path := filepath.Join(subpath...)
	path = filepath.Join(parent, path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return failures.FailIO.Wrap(err)
		}
	}
	return nil
}
