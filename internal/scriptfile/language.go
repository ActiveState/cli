package scriptfile

import (
	"fmt"
	"strings"
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

var lookup = [...]string{
	"unknown",
	"bash",
	"sh",
	"batch",
	"perl",
	"python2",
	"python3",
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
		if strings.ToLower(name) == v {
			return Language(i)
		}
	}
	return Unknown
}

func (l *Language) String() string {
	if int(*l) < 0 || int(*l) > len(lookup)-1 {
		return lookup[0]
	}
	return lookup[*l]
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
