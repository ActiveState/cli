package fileutils

import (
	"bytes"
	"errors"
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
func replacePathInFile(buf []byte, oldpath, newpath string) (bool, []byte, error) {
	if IsBinary(buf) {
		return replaceNulTerminatedPath(buf, oldpath, newpath)
	}
	return replacePathInTextFile(buf, oldpath, newpath)
}

func replacePathInTextFile(buf []byte, oldpath, newpath string) (bool, []byte, error) {
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
		return false, buf, nil
	}
	res.Write(buf[lw:])
	return true, res.Bytes(), nil
}

func replaceNulTerminatedPath(buf []byte, oldpath, newpath string) (bool, []byte, error) {
	findBytes := []byte(oldpath)
	replaceBytes := []byte(newpath)
	if len(findBytes) < len(replaceBytes) {
		return false, nil, errors.New("replacement text cannot be longer than search text in a binary file")
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
	return count > 0, buf, nil
}
