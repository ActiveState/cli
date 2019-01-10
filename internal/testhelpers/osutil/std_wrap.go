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

func replaceStdin(newIn *os.File) *os.File {
	oldIn := os.Stdin
	os.Stdin = newIn
	return oldIn
}

// oldStdin := os.Stdin
// tmpIn, inWriter, _ := os.Pipe()
// defer func() { os.Stdin = oldStdin }()
// os.Stdin = tmpIn

// cmd.Config().GetCobraCmd().SetArgs([]string{"generate", "-b", "512"})
// inWriter.Write([]byte("abc123\n"))
// execErr = cmd.Config().Execute()

// CaptureStdout will execute a provided function and capture anything written to stdout.
// It will then return that output as a string along with any errors captured in the process.
func CaptureStdout(fnToExec func()) (string, error) {
	outReader, tmpOut, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer replaceStdout(replaceStdout(tmpOut))

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

// WrapStdin will buffer stdin with the lines provided as a variadic set of string before
// executing the provided function. Each line will be appended with a newline (\n) only.
func WrapStdin(fnToExec func(), inputLines ...string) {
	tmpIn, inWriter, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	defer inWriter.Close()
	defer tmpIn.Close()

	defer replaceStdin(replaceStdin(tmpIn))

	for _, line := range inputLines {
		inWriter.Write([]byte(line + "\n"))
	}

	fnToExec() // execute the provided function
}
