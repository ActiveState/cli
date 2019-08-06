package scriptfile

import (
	"fmt"
	"io/ioutil"
	"os"
)

// ScriptFile ...
type ScriptFile struct {
	lang   Language
	script string
	file   string
}

// New ...
func New(l Language, script string) (*ScriptFile, error) {
	file, err := createFile(script, tempFileName(l), fileHeader(l))
	if err != nil {
		return nil, err
	}

	sf := ScriptFile{
		lang:   l,
		script: script,
		file:   file,
	}

	return &sf, nil
}

// Clean ...
func (sf *ScriptFile) Clean() {
	os.Remove(sf.file)
}

// FileName ...
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

	if err := os.Chmod(f.Name(), 0755); err != nil {
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
