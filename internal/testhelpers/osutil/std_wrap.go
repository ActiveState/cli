package osutil

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
)

func replaceStderr(newErr *os.File) *os.File {
	oldErr := os.Stderr
	os.Stderr = newErr
	return oldErr
}

func replaceStdout(newOut *os.File) *os.File {
	oldOut := os.Stdout
	os.Stdout = newOut
	return oldOut
}

func replaceStdin(newIn *os.File) *os.File {
	oldIn := os.Stdin
	os.Stdin = newIn
	return oldIn
}

// CaptureStderr will execute a provided function and capture anything written to stderr.
// It will then return that output as a string along with any errors captured in the process.
func CaptureStderr(fnToExec func()) (string, error) {
	errReader, tmpErr, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer replaceStderr(replaceStderr(tmpErr))

	fnToExec() // execute the provided function

	if err = tmpErr.Close(); err != nil {
		return "", err
	}

	errBytes, err := ioutil.ReadAll(errReader)
	errStr := string(errBytes)
	if err != nil {
		err = errReader.Close()
	}
	return errStr, err
}

// CaptureStdout will execute a provided function and capture anything written to stdout.
// It will then return that output as a string along with any errors captured in the process.
func CaptureStdout(fnToExec func()) (string, error) {
	outReader, tmpOut, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer replaceStdout(replaceStdout(tmpOut))

	// Redefine output used for color printing, otherwise these won't be captured
	color.Output = colorable.NewColorableStdout()
	defer func() { color.Output = colorable.NewColorableStdout() }()

	fnToExec() // execute the provided function

	if err = tmpOut.Close(); err != nil {
		return "", err
	}

	outBytes, err := ioutil.ReadAll(outReader)
	outStr := string(outBytes)
	if err != nil {
		err = outReader.Close()
	}
	return outStr, err
}

// WrapStdin will fill stdin with the lines provided as a variadic list of strings before
// executing the provided function. Each line will be appended with a newline (\n) only.
func WrapStdin(fnToExec func(), inputLines ...string) {
	WrapStdinWithDelay(0, fnToExec, inputLines...)
}

// WrapStdinWithDelay will fill stdin with the lines provided, but with a given delay before
// each write. This is useful if there is a reader that reads all of stdin between prompts,
// for instance.
func WrapStdinWithDelay(delay time.Duration, fnToExec func(), inputLines ...string) {
	tmpIn, inWriter, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	defer tmpIn.Close()
	defer replaceStdin(replaceStdin(tmpIn))

	if delay > 0 {
		// need to run this asynchornously so that the fnToExec can be processed
		go writeLinesAndClosePipe(inWriter, inputLines, func() { time.Sleep(delay) })
	} else {
		writeLinesAndClosePipe(inWriter, inputLines, nil)
	}

	fnToExec() // execute the provided function
}

func writeLinesAndClosePipe(writer *os.File, lines []string, callbackFn func()) {
	defer writer.Close()
	for _, line := range lines {
		if callbackFn != nil {
			callbackFn()
		}
		writer.WriteString(line + "\n")
	}
}
