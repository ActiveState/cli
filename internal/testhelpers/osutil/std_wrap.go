package osutil

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
)

func replaceStderr(newErr *os.File) *os.File {
	oldErr := os.Stderr
	os.Stderr = newErr
	color.Output = newErr
	return oldErr
}

func replaceStdout(newOut *os.File) *os.File {
	oldOut := os.Stdout
	os.Stdout = newOut
	color.Output = newOut
	return oldOut
}

func replaceStdin(newIn *os.File) *os.File {
	oldIn := os.Stdin
	os.Stdin = newIn
	return oldIn
}

// captureWrites will execute a provided function and return any bytes written to the provided
// writer from the provided reader (assuming they are generated from something like an os.Pipe).
func captureWrites(fnToExec func(), reader, writer *os.File) (string, error) {
	fnToExec() // execute the provided function

	if err := writer.Close(); err != nil {
		return "", err
	}

	writeBytes, err := io.ReadAll(reader)
	if err != nil {
		err = reader.Close()
	}
	return string(writeBytes), err
}

// CaptureStderr will execute a provided function and capture anything written to stderr.
// It will then return that output as a string along with any errors captured in the process.
func CaptureStderr(fnToExec func()) (string, error) {
	errReader, errWriter, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer replaceStderr(replaceStderr(errWriter))
	return captureWrites(fnToExec, errReader, errWriter)
}

// CaptureStdout will execute a provided function and capture anything written to stdout.
// It will then return that output as a string along with any errors captured in the process.
func CaptureStdout(fnToExec func()) (string, error) {
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer replaceStdout(replaceStdout(outWriter))
	return captureWrites(fnToExec, outReader, outWriter)
}

// WrapStdin will fill stdin with the lines provided as a variadic list of strings before
// executing the provided function. Each line will be appended with a newline (\n) only.
func WrapStdin(fnToExec func(), inputLines ...interface{}) {
	WrapStdinWithDelay(0, fnToExec, inputLines...)
}

// WrapStdinWithDelay will fill stdin with the lines provided, but with a given delay before
// each write. This is useful if there is a reader that reads all of stdin between prompts,
// for instance.
func WrapStdinWithDelay(delay time.Duration, fnToExec func(), inputLines ...interface{}) {
	tmpIn, inWriter, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	defer tmpIn.Close()
	defer replaceStdin(replaceStdin(tmpIn))

	if delay > 0 {
		// need to run this asynchornously so that the fnToExec can be processed
		go func() {
			time.Sleep(500 * time.Millisecond) // Give fnToExec some time to start
			writeLinesAndClosePipe(inWriter, inputLines, func() { time.Sleep(delay) })
		}()
	} else {
		writeLinesAndClosePipe(inWriter, inputLines, nil)
	}

	fnToExec() // execute the provided function
}

func writeLinesAndClosePipe(writer *os.File, lines []interface{}, callbackFn func()) {
	defer writer.Close()
	for _, line := range lines {
		if callbackFn != nil {
			callbackFn()
		}
		if lineStr, ok := line.(string); ok {
			_, err := writer.WriteString(lineStr + "\n")
			if err != nil {
				log.Panicf("Error writing to stdin: %v", err)
			}
		} else if lineRune, ok := line.(rune); ok {
			_, err := writer.WriteString(string(lineRune))
			if err != nil {
				log.Panicf("Error writing to stdin: %v", err)
			}
		} else {
			log.Panicf("Unsupported line: %v", line)
		}
	}
}
