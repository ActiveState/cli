package artefacts

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"

	"github.com/ActiveState/ActiveState-CLI/internal/logging"
)

// ReplaceAll replaces all instances of search text with replacement text in a
// file, which may be a binary file.
func ReplaceAll(filename, find, replace string) error {
	// Read the artefact's bytes and create find and replace byte arrays for
	// search and replace.
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	findBytes := []byte(find)
	replaceBytes := []byte(replace)

	// Check if the artefact is a binary file. If so, the search and replace byte
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

	// Open a temporary file for the replacement artefact and then perform the
	// search and replace.
	tmpfile, err := ioutil.TempFile("", "activestatecli-artefact")
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

	// Replace the original artefact.
	stat, _ := os.Stat(filename)
	if err := os.Chmod(tmpfile.Name(), stat.Mode()); err != nil {
		return err
	}
	return os.Rename(tmpfile.Name(), filename)
}
