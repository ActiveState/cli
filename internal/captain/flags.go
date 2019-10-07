package captain

import "github.com/ActiveState/cli/internal/failures"

type FlagType int

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
	Type        FlagType
	Persist     bool

	OnUse func()

	StringVar   *string
	StringValue string
	IntVar      *int
	IntValue    int
	BoolVar     *bool
	BoolValue   bool
}

func (c *Command) setFlags(flags []*Flag) error {
	c.flags = flags
	for _, flag := range flags {
		flagSetter := c.cobra.Flags
		if flag.Persist {
			flagSetter = c.cobra.PersistentFlags
		}

		switch flag.Type {
		case TypeString:
			flagSetter().StringVarP(flag.StringVar, flag.Name, flag.Shorthand, flag.StringValue, flag.Description)
		case TypeInt:
			flagSetter().IntVarP(flag.IntVar, flag.Name, flag.Shorthand, flag.IntValue, flag.Description)
		case TypeBool:
			flagSetter().BoolVarP(flag.BoolVar, flag.Name, flag.Shorthand, flag.BoolValue, flag.Description)
		default:
			return failures.FailInput.New("Unknown type:" + string(flag.Type))
		}
	}

	return nil
}
