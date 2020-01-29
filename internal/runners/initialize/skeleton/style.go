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
		i = 1
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

// Recognized returns whether the style is a known useful value.
func (s *Style) Recognized() bool {
	return s != nil && *s != Unset && *s != Unknown
}

// Set implements the captain marshaler interfaces.
func (s *Style) Set(v string) error {
	if s == nil {
		return fmt.Errorf("cannot set nil style")
	}

	style := MakeStyleByName(v)
	if !style.Recognized() {
		names := RecognizedStylesNames()

		return fmt.Errorf(locale.Tr(
			"err_invalid_skeleton_style", v, strings.Join(names, ", "),
		))
	}

	*s = style
	return nil
}

// Type implements the captain.FlagMarshaler interface.
func (s *Style) Type() string {
	return "skeleton-style"
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
