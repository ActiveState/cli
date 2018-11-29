package osutil

import (
	"io/ioutil"
	"os"
)

func replaceStdout(newOut *os.File) *os.File {
	oldOut := os.Stdout
	os.Stdout = newOut
	return oldOut
}

// CaptureStdout will execute a provided function and capture anything written to stdout.
// It will then return that output as a string along with any errors captured in the process.
func CaptureStdout(fnToExec func()) (string, error) {
	tmpReader, tmpOut, err := os.Pipe()
	if err != nil {
		return "", err
	}

	osStdout := replaceStdout(tmpOut)
	defer replaceStdout(osStdout)

	fnToExec() // execute the provided function

	if err = tmpOut.Close(); err != nil {
		return "", err
	}

	outBytes, err := ioutil.ReadAll(tmpReader)
	outStr := string(outBytes)
	if err != nil {
		err = tmpReader.Close()
	}
	return outStr, err
}
