package captain

import (
	"reflect"

	"github.com/ActiveState/cli/internal/failures"
)

type FlagType int

type FlagMarshaler interface {
	String() string
	Set(string) error
	Type() string
}

// Note we only support the types that we currently have need for. You can add more as needed. Check the pflag docs
// for reference: https://godoc.org/github.com/spf13/pflag
const (
	// TypeString is used to define the type for flags/args
	TypeString FlagType = iota
	// TypeInt is used to define the type for flags/args
	TypeInt
	// TypeBool is used to define the type for flags/args
	TypeBool
)

// Flag is used to define flags in our Command struct
type Flag struct {
	Name        string
	Shorthand   string
	Description string
	Persist     bool
	Value       interface{}

	OnUse func()
}

func (c *Command) setFlags(flags []*Flag) error {
	c.flags = flags
	for _, flag := range flags {
		flagSetter := c.cobra.Flags
		if flag.Persist {
			flagSetter = c.cobra.PersistentFlags
		}

		if flag.Value != nil {
			switch v := flag.Value.(type) {
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
				return failures.FailInput.New(
					"Unknown type:" + reflect.TypeOf(v).Name(),
				)
			}
		}
	}

	return nil
}
