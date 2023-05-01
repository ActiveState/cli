package captain

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
)

type FlagMarshaler interface {
	String() string
	Set(string) error
	Type() string
}

// Flag is used to define flags in our Command struct
type Flag struct {
	Name        string
	Shorthand   string
	Description string
	Persist     bool
	Value       interface{}
	Hidden      bool

	OnUse func()
}

func (c *Command) setFlags(flags []*Flag) error {
	c.flags = flags
	for _, flag := range flags {
		flagSetter := c.cobra.Flags
		if flag.Persist {
			flagSetter = c.cobra.PersistentFlags
		}

		switch v := flag.Value.(type) {
		case nil:
			return errs.New("flag value must not be nil (%v)", flag)
		case *string:
			flagSetter().StringVarP(
				v, flag.Name, flag.Shorthand, *v, flag.Description,
			)
		case *int:
			flagSetter().IntVarP(
				v, flag.Name, flag.Shorthand, *v, flag.Description,
			)
		case *bool:
			flagSetter().BoolVarP(
				v, flag.Name, flag.Shorthand, *v, flag.Description,
			)
		case FlagMarshaler:
			flagSetter().VarP(
				v, flag.Name, flag.Shorthand, flag.Description,
			)
		default:
			return errs.New(
				fmt.Sprintf("Unknown type: %s (%v)"+reflect.TypeOf(v).Name(), v),
			)
		}

		if flag.Hidden {
			if err := flagSetter().MarkHidden(flag.Name); err != nil {
				return errs.Wrap(err, "markFlagHidden %s failed", flag.Name)
			}
		}
	}

	return nil
}

type NullString struct {
	s     string
	isSet bool
}

func (s *NullString) String() string {
	if s.isSet {
		return s.s
	}
	return ""
}

func (s *NullString) Set(v string) error {
	s.s = v
	s.isSet = true
	return nil
}

func (s *NullString) Type() string {
	return "null string"
}

func (s *NullString) IsSet() bool {
	return s.isSet
}

func (s *NullString) AsPtrTo() *string {
	if !s.isSet {
		return nil
	}
	v := s.s
	return &v
}

type NullInt struct {
	n     int
	isSet bool
}

func (n *NullInt) String() string {
	if n.isSet {
		return strconv.FormatInt(int64(n.n), 10)
	}
	return ""
}

func (n *NullInt) Set(v string) error {
	x, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("null int set: %w", err)
	}

	n.n = x
	n.isSet = true

	return nil
}

func (n *NullInt) Type() string {
	return "null int"
}

func (n *NullInt) IsSet() bool {
	return n.isSet
}

func (n *NullInt) AsPtrTo() *int {
	if !n.isSet {
		return nil
	}
	v := n.n
	return &v
}

type NullBool struct {
	b     bool
	isSet bool
}

func (b *NullBool) String() string {
	if b.isSet {
		return fmt.Sprintf("%t", b.b)
	}
	return ""
}

func (b *NullBool) Set(v string) error {
	b.b = strings.ToLower(v) == "true"
	if !b.b && strings.ToLower(v) != "false" {
		return fmt.Errorf("null bool set: %q is not a valid value", v)
	}
	b.isSet = true
	return nil
}

func (b *NullBool) Type() string {
	return "null bool"
}

func (b *NullBool) IsSet() bool {
	return b.isSet
}

func (b *NullBool) AsPtrTo() *bool {
	if !b.isSet {
		return nil
	}
	v := b.b
	return &v
}
