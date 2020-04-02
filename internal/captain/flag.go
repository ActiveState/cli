package captain

import (
	"reflect"

	"github.com/ActiveState/cli/internal/failures"
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
			return failures.FailDeveloper.New("flag value must not be nil")
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
			return failures.FailDeveloper.New(
				"Unknown type:" + reflect.TypeOf(v).Name(),
			)
		}

		if flag.Hidden {
			c.markFlagHidden(flag.Name)
		}
	}

	return nil
}
