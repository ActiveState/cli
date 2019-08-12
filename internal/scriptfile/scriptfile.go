package scriptfile

import (
	"fmt"
	"io/ioutil"
	"os"
)

// ScriptFile represents an on-disk executable file.
type ScriptFile struct {
	lang Language
	file string
}

// New receives a language and script body that are used to construct a runable
// on-disk file that is tracked by the returned value.
func New(l Language, script string) (*ScriptFile, error) {
	file, err := createFile(script, tempFileName(l), fileHeader(l))
	if err != nil {
		return nil, err
	}

	sf := ScriptFile{
		lang: l,
		file: file,
	}

	return &sf, nil
}

// Clean provides simple/deferable clean up.
func (sf *ScriptFile) Clean() {
	os.Remove(sf.file)
}

// FileName returns the on-disk filename of the tracked script file.
func (sf *ScriptFile) FileName() string {
	return sf.file
}

func createFile(script, name, header string) (string, error) {
	f, err := ioutil.TempFile("", name)
	if err != nil {
		return "", err
	}

	if _, err = f.WriteString(header + script); err != nil {
		return "", err
	}

	if err = f.Close(); err != nil {
		return "", err
	}

	if err := os.Chmod(f.Name(), 0700); err != nil {
		return "", err
	}

	return f.Name(), nil
}

func tempFileName(l Language) string {
	namePrefix := "script-*"

	switch l {
	case Batch:
		return namePrefix + ".bat"
	default:
		return namePrefix + ".tmp"
	}
}

func fileHeader(l Language) string {
	switch l {
	case Batch, Unknown:
		return ""
	default:
		return fmt.Sprintf("#!/usr/bin/env %s\n", l.String())
	}
}
