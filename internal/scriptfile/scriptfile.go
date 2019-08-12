package scriptfile

import (
	"os"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

// ScriptFile represents an on-disk executable file.
type ScriptFile struct {
	lang Language
	file string
}

// New receives a language and script body that are used to construct a runable
// on-disk file that is tracked by the returned value.
func New(l Language, script string) (*ScriptFile, *failures.Failure) {
	file, fail := fileutils.CreateTempExecutable("", l.TempPattern(), l.Header()+script)
	if fail != nil {
		return nil, fail
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

// Filename returns the on-disk filename of the tracked script file.
func (sf *ScriptFile) Filename() string {
	return sf.file
}
