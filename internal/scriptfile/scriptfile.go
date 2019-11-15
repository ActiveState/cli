package scriptfile

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
)

// ScriptFile represents an on-disk executable file.
type ScriptFile struct {
	lang language.Language
	file string
}

// New receives a language and script body that are used to construct a runable
// on-disk file that is tracked by the returned value.
func New(l language.Language, name, script string) (*ScriptFile, *failures.Failure) {
	return new(l, name, []byte(l.Header()+script))
}

// NewAsSource recieves a language and script body that are used to construct an
// on-disk file that is tracked by the return value. This file is not guaranteed
// to be runnable
func NewAsSource(l language.Language, name, script string) (*ScriptFile, *failures.Failure) {
	return new(l, name, []byte(script))
}

func new(l language.Language, name string, script []byte) (*ScriptFile, *failures.Failure) {
	file, fail := fileutils.WriteTempFile(
		"", fmt.Sprintf("%s*%s", name, l.Ext()), []byte(script), 0700,
	)
	if fail != nil {
		return nil, fail
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
