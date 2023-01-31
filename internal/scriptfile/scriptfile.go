package scriptfile

import (
	"fmt"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"os"
)

// ScriptFile represents an on-disk executable file.
type ScriptFile struct {
	lang language.Language
	file string
}

// New receives a language and script body that are used to construct a runable
// on-disk file that is tracked by the returned value.
func New(l language.Language, name, script string) (*ScriptFile, error) {
	return new(l, name, []byte(l.Header()+script))
}

// NewEmpty receives a language that is used to construct a runnable, but empty,
// on-disk file that is tracked by the return value.
func NewEmpty(l language.Language, name string) (*ScriptFile, error) {
	return new(l, name, []byte(""))
}

// NewAsSource recieves a language and script body that are used to construct an
// on-disk file that is tracked by the return value. This file is not guaranteed
// to be runnable
func NewAsSource(l language.Language, name, script string) (*ScriptFile, error) {
	return new(l, name, []byte(script))
}

func new(l language.Language, name string, script []byte) (*ScriptFile, error) {
	file, err := fileutils.WriteTempFileToDir(
		"", fmt.Sprintf("%s*%s", name, l.Ext()), []byte(script), 0700,
	)
	if err != nil {
		return nil, err
	}

	return &ScriptFile{
		lang: l,
		file: file,
	}, nil
}

// Clean provides simple/deferable clean up.
func (sf *ScriptFile) Clean() {
	os.Remove(sf.file)
}

// Filename returns the on-disk filename of the tracked script file.
func (sf *ScriptFile) Filename() string {
	return sf.file
}

// Write updates the on-disk scriptfile with the script value
func (sf *ScriptFile) Write(value string) error {
	return fileutils.WriteFile(sf.file, []byte(sf.lang.Header()+value))
}
