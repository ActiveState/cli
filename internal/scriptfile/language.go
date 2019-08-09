package scriptfile

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
)

// Language ...
type Language int

// ...
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

func ptrToStr(s string) *string {
	return &s
}

// MakeLanguageByShell ...
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

func (l *Language) String() string {
	if int(*l) < 0 || int(*l) > len(lookup)-1 {
		return lookup[0].name
	}
	return lookup[*l].name
}

// Executable ...
func (l *Language) Executable() Executable {
	if int(*l) < 0 || int(*l) > len(lookup)-1 {
		return lookup[0].exec
	}
	return lookup[*l].exec
}

// UnmarshalYAML ...
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

// MarshalYAML ...
func (l *Language) MarshalYAML() (interface{}, error) {
	return l.String(), nil
}

// Executable ...
type Executable struct {
	name string
	base bool
}

// Name ...
func (e *Executable) Name() string {
	return e.name
}

// Builtin ...
func (e *Executable) Builtin() bool {
	return e.base
}
