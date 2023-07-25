package hello

import (
	"github.com/ActiveState/cli/internal/errs"
)

// Text represents an implementation of captain.FlagMarshaler. It is a trivial
// example, and does not do very much interesting except provide a way to check
// if it has been set by the user.
type Text struct {
	Value string
	isSet bool
}

// Set implements the captain flagmarshaler interface.
func (t *Text) Set(v string) error {
	if t == nil {
		return errs.New("cannot set nil value")
	}
	t.Value, t.isSet = v, true
	return nil
}

// String implements the fmt.Stringer and flagmarshaler interfaces.
func (t *Text) String() string {
	if t == nil {
		return ""
	}
	return t.Value
}

// Type implements the flagmarshaler interface.
func (t *Text) Type() string {
	return "text"
}

func (ns *Text) IsSet() bool {
	return ns.isSet
}
