package captain

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
