package captain

type ArgMarshaler interface {
	Set(string) error
}

type ArgMarshalerVariable interface {
	Set(...string) error
}

// Argument is used to define flags in our Command struct
type Argument struct {
	Name           string
	Description    string
	Required       bool
	Value          interface{}
	VariableLength bool // If true; consumes all remaining input arguments. MUST be the last defined command argument.
}
