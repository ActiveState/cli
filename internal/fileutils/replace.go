package fileutils

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
)

// checkPathMatch tries to find `findBytes` at the beginning of `buf` and returns the length of the match.
// It matches multiple backslashes with a single backslash
// If no match could be found -1 is returned
func checkPathMatch(buf []byte, findBytes []byte) int {
	i := 0
	for ; i < len(findBytes); i++ {
		b := buf[i]
		if b == '\\' && i+1 < len(buf) && buf[i+1] == '\\' {
			continue
		}
		if b != findBytes[i] {
			return -1
		}
	}
	return i
}

// replacePathInFile replaces all occurrences of oldpath with newpath
// For binary files with nul-terminated strings, it ensures that the replaces strings are still valid nul-terminated strings and the returned buffer has the same size as the input buffer buf
// The first return argument denotes the number of replacements.
func replacePathInFile(buf []byte, oldpath, newpath string) (int, []byte, error) {
	if IsBinary(buf) {
		return replaceNulTerminatedPath(buf, oldpath, newpath)
	}
	return replacePathInTextFile(buf, oldpath, newpath)
}

func replacePathInTextFile(buf []byte, oldpath, newpath string) (int, []byte, error) {
	findBytes := []byte(oldpath)
	replaceBytes := []byte(newpath)

	res := bytes.NewBuffer(make([]byte, 0, len(buf)))

	count := 0 // number of replacements
	lw := 0    // index of last write to result buffer
	for i := 0; ; i++ {
		// find first byte of oldpath in buffer
		offset := bytes.IndexByte(buf[i:], findBytes[0])
		if offset < 0 { // reached the end of the buffer
			break
		}
		// update index
		i += offset

		// check if we can match the entire oldpath from here
		j := checkPathMatch(buf[i:], findBytes)
		if j < 0 {
			continue
		}
		count++

		// write replaced text to result buffer
		startOffset := i
		endOffset := i + j

		// write everything from last write position to start
		res.Write(buf[lw:startOffset])
		// write replacement
		res.Write(replaceBytes)
		// update lw position
		lw = endOffset
	}
	// if nothing was replaced, we just return the initial buffer
	if count == 0 {
		return 0, buf, nil
	}
	res.Write(buf[lw:])
	return count, res.Bytes(), nil
}

func replaceNulTerminatedPath(buf []byte, oldpath, newpath string) (int, []byte, error) {
	findBytes := []byte(oldpath)
	replaceBytes := []byte(newpath)
	if len(findBytes) < len(replaceBytes) {
		return -1, nil, errors.New("replacement text cannot be longer than search text in a binary file")
	}
	zeros := make([]byte, len(findBytes)-len(replaceBytes))

	count := 0 // number of replacements
	for i := 0; ; i++ {
		// find first byte of oldpath in buffer
		offset := bytes.IndexByte(buf[i:], findBytes[0])
		if offset < 0 {
			break
		}
		// update index
		i += offset

		// check if we can match the entire oldpath from here
		j := checkPathMatch(buf[i:], findBytes)
		if j < 0 {
			continue
		}

		count++

		startOffset := i
		midOffset := i + j

		// find zero byte at end of string
		j = bytes.IndexByte(buf[midOffset:], nullByte)
		endOffset := midOffset + j

		// modify the buffer in place:
		// add the replacement text
		no := copy(buf[startOffset:], replaceBytes)
		// move rest of string forward
		no2 := copy(buf[startOffset+no:], buf[midOffset:endOffset])
		// add zeros
		copy(buf[startOffset+no+no2:], zeros)

		// fast forward to the end of the string
		i = endOffset - 1
	}
	return count, buf, nil
}

// replacePathInFileRegex is the old implementation of replacePathInFile based on regular expressions.
// It is now only used in the benchmark functions
// Note that it always returns 1 for the count of replacements
func replacePathInFileRegex(buf []byte, oldpath, newpath string) (int, []byte, error) {
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
			return -1, nil, errors.New("replacement text cannot be longer than search text in a binary file")
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
	buffer := bytes.NewBuffer([]byte{})
	buffer.Write(replaced)

	return 1, buffer.Bytes(), nil
}
