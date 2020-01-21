package language

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
)

// Language tracks the languages potentially used.
type Language int

// Language constants are provided for safety/reference.
const (
	Unset Language = iota
	Unknown
	Bash
	Sh
	Batch
	Perl
	Python2
	Python3
)

const (
	filePatternPrefix = "script-*"
)

type languageData struct {
	name    string
	text    string
	ext     string
	hdr     bool
	require string
	version string
	exec    Executable
}

var lookup = [...]languageData{
	{},
	{
		"unknown", locale.T("language_name_unknown"), ".tmp", false, "", "",
		Executable{"", false},
	},
	{
		"bash", "Bash", ".sh", true, "", "",
		Executable{"", true},
	},
	{
		"sh", "Shell", ".sh", true, "", "",
		Executable{"", true},
	},
	{
		"batch", "Batch", ".bat", false, "", "",
		Executable{"", true},
	},
	{
		"perl", "Perl", ".pl", true, "perl", "5.28.1",
		Executable{constants.ActivePerlExecutable, false},
	},
	{
		"python2", "Python 2", ".py", true, "python", "2.7.14",
		Executable{constants.ActivePython2Executable, false},
	},
	{
		"python3", "Python 3", ".py", true, "python", "3.6.6",
		Executable{constants.ActivePython3Executable, false},
	},
}

// MakeByShell returns either bash or cmd based on whether the provided
// shell name contains "cmd". This should be taken to mean that bash is a sort
// of default.
func MakeByShell(shell string) Language {
	shell = strings.ToLower(shell)

	if strings.Contains(shell, "cmd") {
		return Batch
	}

	return Bash
}

// MakeByName will retrieve a language by a given name after lower-casing.
func MakeByName(name string) Language {
	for i, data := range lookup {
		if strings.ToLower(name) == data.name {
			return Language(i)
		}
	}

	return Unknown
}

func (l Language) data() languageData {
	i := int(l)
	if i < 0 || i > len(lookup)-1 {
		i = 0
	}
	return lookup[i]
}

// String implements the fmt.Stringer interface.
func (l *Language) String() string {
	if l == nil {
		return ""
	}
	return l.data().name
}

// Text returns the human-readable value.
func (l *Language) Text() string {
	if l == nil {
		return ""
	}
	return l.data().text
}

// Recognized returns whether the language is a known useful value.
func (l *Language) Recognized() bool {
	return l != nil && *l != Unset && *l != Unknown
}

// Ext return the file extension for the language.
func (l Language) Ext() string {
	return l.data().ext
}

// Header returns the interpreter directive.
func (l Language) Header() string {
	ld := l.data()
	if ld.hdr {
		return fmt.Sprintf("#!/usr/bin/env %s\n", ld.name)
	}
	return ""
}

// TempPattern returns the ioutil.TempFile pattern to be used to form the temp
// file name.
func (l Language) TempPattern() string {
	return filePatternPrefix + l.data().ext
}

// Requirement returns the platform-level string representation.
func (l Language) Requirement() string {
	return l.data().require
}

// RecommendedVersion returns the string representation of the recommended
// version.
func (l Language) RecommendedVersion() string {
	return l.data().version
}

// Executable provides details about the executable related to the Language.
func (l Language) Executable() Executable {
	return l.data().exec
}

// UnmarshalYAML implements the go-yaml/yaml.Unmarshaler interface.
func (l *Language) UnmarshalYAML(applyPayload func(interface{}) error) error {
	if l == nil {
		return fmt.Errorf("cannot unmarshal to nil language")
	}

	var payload string
	if err := applyPayload(&payload); err != nil {
		return err
	}

	return l.Set(payload)
}

// MarshalYAML implements the go-yaml/yaml.Marshaler interface.
func (l Language) MarshalYAML() (interface{}, error) {
	return l.String(), nil
}

// Set implements the captain marshaler interfaces.
func (l *Language) Set(v string) error {
	if l == nil {
		return fmt.Errorf("cannot set nil language")
	}

	lang := MakeByName(v)
	if !lang.Recognized() {
		names := RecognizedNames()

		return fmt.Errorf(locale.Tr(
			"err_invalid_language", v, strings.Join(names, ", "),
		))
	}

	*l = lang
	return nil
}

// Type implements the captain.FlagMarshaler interface.
func (l *Language) Type() string {
	return "language"
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

// Available returns whether the executable is not "builtin" and also has a
// defined name.
func (e Executable) Available() bool {
	return !e.base && e.name != ""
}

// Recognized returns all languages that are supported.
func Recognized() []Language {
	var langs []Language
	for i := range lookup {
		if l := Language(i); l.Recognized() {
			langs = append(langs, l)
		}
	}
	return langs
}

// RecognizedNames returns all language names that are supported.
func RecognizedNames() []string {
	var ls []string
	for i, data := range lookup {
		if l := Language(i); l.Recognized() {
			ls = append(ls, data.name)
		}
	}
	return ls
}

// Supported tracks the languages potentially used for projects.
type Supported struct {
	Language
}

// Recognized returns whether the supported language is a known useful value.
func (l *Supported) Recognized() bool {
	if l == nil {
		return false
	}
	return l.Language.Recognized() && l.Executable().Available()
}

// UnmarshalYAML implements the go-yaml/yaml.Unmarshaler interface.
func (l *Supported) UnmarshalYAML(f func(interface{}) error) error {
	if l == nil {
		return fmt.Errorf("cannot unmarshal to nil supported language")
	}

	var s string
	if err := f(&s); err != nil {
		return err
	}

	return l.Set(s)
}

// Set implements the captain marshaler interfaces.
func (l *Supported) Set(v string) error {
	if l == nil {
		return fmt.Errorf("cannot set nil supported language")
	}

	lbn := Supported{MakeByName(v)}
	if !lbn.Recognized() {
		names := RecognizedSupportedsNames()

		return fmt.Errorf(locale.Tr(
			"err_invalid_language", v, strings.Join(names, ", "),
		))
	}

	*l = lbn
	return nil
}

// RecognizedSupporteds returns all languages that are not "builtin"
// and also have a defined executable name.
func RecognizedSupporteds() []Supported {
	var ls []Supported
	for i := range lookup {
		l := Supported{Language(i)}
		if l.Recognized() {
			ls = append(ls, l)
		}
	}
	return ls
}

// RecognizedSupportedsNames returns all languages that are not
// "builtin" and also have a defined executable name.
func RecognizedSupportedsNames() []string {
	var ls []string
	for i, v := range lookup {
		l := Supported{Language(i)}
		if l.Recognized() {
			ls = append(ls, v.name)
		}
	}
	return ls
}
