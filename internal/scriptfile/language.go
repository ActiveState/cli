package scriptfile

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
)

// Language tracks the languages potentially used for scripts.
type Language int

// Language constants are provided for safety/reference.
const (
	Unknown Language = iota
	Bash
	Sh
	Batch
	Perl
	Python2
	Python3
)

type languageData struct {
	name string
	exec Executable
}

var lookup = [...]languageData{
	{"unknown", Executable{"", false}},
	{"bash", Executable{"", true}},
	{"sh", Executable{"", true}},
	{"batch", Executable{"", true}},
	{"perl", Executable{constants.ActivePerlExecutable, false}},
	{"python2", Executable{constants.ActivePython2Executable, false}},
	{"python3", Executable{constants.ActivePython3Executable, false}},
}

// MakeLanguageByShell returns either bash or cmd based on whether the provided
// shell name contains "cmd". This should be taken to mean that bash is a sort
// of default.
func MakeLanguageByShell(shell string) Language {
	shell = strings.ToLower(shell)

	if strings.Contains(shell, "cmd") {
		return Batch
	}

	return Bash
}

func makeLanguage(name string) Language {
	for i, v := range lookup {
		if strings.ToLower(name) == v.name {
			return Language(i)
		}
	}
	return Unknown
}

// String implements the fmt.Stringer interface.
func (l *Language) String() string {
	i := int(*l)
	if i < 0 || i > len(lookup)-1 {
		i = 0
	}
	return lookup[i].name
}

// Executable provides details about the executable related to the Language.
func (l Language) Executable() Executable {
	i := int(l)
	if i < 0 || i > len(lookup)-1 {
		i = 0
	}
	return lookup[i].exec
}

// UnmarshalYAML implements the go-yaml/yaml.Unmarshaler interface.
func (l *Language) UnmarshalYAML(f func(interface{}) error) error {
	var s string
	if err := f(&s); err != nil {
		return err
	}

	*l = makeLanguage(s)

	if len(s) > 0 && *l == Unknown {
		return fmt.Errorf("cannot unmarshal yaml")
	}

	return nil
}

// MarshalYAML implements the go-yaml/yaml.Marshaler interface.
func (l *Language) MarshalYAML() (interface{}, error) {
	return l.String(), nil
}

// Executable contains details about an executable program used to interpret a
// Language.
type Executable struct {
	name string
	base bool
}

// Name returns the executables file's name.
func (e Executable) Name() string {
	return e.name
}

// Builtin expresses whether the executable is expected to be provided by the
// shell environment.
func (e Executable) Builtin() bool {
	return e.base
}
