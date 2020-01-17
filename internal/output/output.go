package output

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

// Format tracks format types
type Format int

// Format constants are provided for safety/reference.
const (
	Unset Format = iota
	Unknown
	JSON
	EditorV0 // Komodo
)

type formatData struct {
	name string
	text string
}

var formatLookup = [...]formatData{
	{},
	{"unknown", "Unknown"},
	{"json", "JSON"},
	{"editor.v0", "Editor V0"},
}

// MakeFormatByName will retrieve a format by a given name
func MakeFormatByName(name string) Format {
	for i, v := range formatLookup {
		if strings.ToLower(name) == v.name {
			return Format(i)
		}
	}

	return Unknown
}

func (f Format) data() formatData {
	i := int(f)
	if i < 0 || i > len(formatLookup)-1 {
		i = 0
	}
	return formatLookup[i]
}

// String implements the fmt.Stringer interface.
func (f *Format) String() string {
	if f == nil {
		return ""
	}
	return f.data().name
}

// Text returns the human-readable value.
func (f *Format) Text() string {
	if f == nil {
		return ""
	}
	return f.data().text
}

// Recognized returns whether the format is a known useful value.
func (f *Format) Recognized() bool {
	return f != nil && *f != Unset && *f != Unknown
}

// UnmarshalYAML implements the go-yaml/yaml.Unmarshaler interface.
func (f *Format) UnmarshalYAML(fn func(interface{}) error) error {
	if f == nil {
		return fmt.Errorf("cannot unmarshal to nil format")
	}

	var v string
	if err := fn(&v); err != nil {
		return err
	}

	return f.Set(v)
}

// MarshalYAML implements the go-yaml/yaml.Marshaler interface.
func (f Format) MarshalYAML() (interface{}, error) {
	return f.String(), nil
}

// Set implements the captain marshaler interfaces.
func (f *Format) Set(v string) error {
	if f == nil {
		return fmt.Errorf("cannot set nil format")
	}

	fbn := MakeFormatByName(v)
	if !fbn.Recognized() {
		names := AvailableFormatsNames()

		return fmt.Errorf(locale.Tr(
			"err_invalid_output_format", v, strings.Join(names, ", "),
		))
	}

	*f = fbn
	return nil
}

// Type implements the captain.FlagMarshaler interface.
func (f *Format) Type() string {
	return "format"
}

// AvailableFormats returns all formats that are supported.
func AvailableFormats() []Format {
	var fs []Format
	for i := range formatLookup {
		if f := Format(i); f.Recognized() {
			fs = append(fs, f)
		}
	}
	return fs
}

// AvailableFormatsNames returns all format names that are supported.
func AvailableFormatsNames() []string {
	var fs []string
	for i, v := range formatLookup {
		if f := Format(i); !f.Recognized() {
			continue
		}
		fs = append(fs, v.name)
	}
	return fs
}
