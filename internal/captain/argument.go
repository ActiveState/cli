package captain

// Argument is used to define flags in our Command struct
type Argument struct {
	Name        string
	Description string
	Required    bool
	Validator   func(arg *Argument, value string) error
	Variable    *string
}
