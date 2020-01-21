package skeleton

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

// Style tracks the styles potentially used for skeletons.
type Style int

// Style constants are provided for safety/reference.
const (
	Unset Style = iota
	Unknown
	Simple
	Editor
)

type styleData struct {
	name string
	text string
}

var styleLookup = [...]styleData{
	{},
	{"unknown", "Unknown"},
	{"simple", "Simple"},
	{"editor", "Editor"},
}

// MakeStyleByName will retrieve a by a given name
func MakeStyleByName(name string) Style {
	for i, data := range styleLookup {
		if strings.ToLower(name) == data.name {
			return Style(i)
		}
	}

	return Unknown
}

func (s Style) data() styleData {
	i := int(s)
	if i < 0 || i > len(styleLookup)-1 {
		i = 0
	}
	return styleLookup[i]
}

// String implements the fmt.Stringer interface.
func (s *Style) String() string {
	if s == nil {
		return ""
	}
	return s.data().name
}

// Text returns the human-readable value.
func (s *Style) Text() string {
	if s == nil {
		return ""
	}
	return s.data().text
}

// Recognized returns whether the style is a known useful value.
func (s *Style) Recognized() bool {
	return s != nil && *s != Unset && *s != Unknown
}

// UnmarshalYAML implements the go-yaml/yaml.Unmarshaler interface.
func (s *Style) UnmarshalYAML(applyPayload func(interface{}) error) error {
	if s == nil {
		return fmt.Errorf("cannot unmarshal to nil style")
	}

	var payload string
	if err := applyPayload(&payload); err != nil {
		return err
	}

	return s.Set(payload)
}

// MarshalYAML implements the go-yaml/yaml.Marshaler interface.
func (s Style) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// Set implements the captain marshaler interfaces.
func (s *Style) Set(v string) error {
	if s == nil {
		return fmt.Errorf("cannot set nil style")
	}

	styleByName := MakeStyleByName(v)
	if !styleByName.Recognized() {
		names := RecognizedStylesNames()

		return fmt.Errorf(locale.Tr(
			"err_invalid_skeleton_style", v, strings.Join(names, ", "),
		))
	}

	*s = styleByName
	return nil
}

// Type implements the captain.FlagMarshaler interface.
func (s *Style) Type() string {
	return "skeleton-style"
}

// RecognizedStyles returns all skeleton styles that are supported.
func RecognizedStyles() []Style {
	var styles []Style
	for i := range styleLookup {
		if s := Style(i); s.Recognized() {
			styles = append(styles, s)
		}
	}
	return styles
}

// RecognizedStylesNames returns all skeleton style names that are supported.
func RecognizedStylesNames() []string {
	var styles []string
	for i, data := range styleLookup {
		if s := Style(i); s.Recognized() {
			styles = append(styles, data.name)
		}
	}
	return styles
}
