package captain

type ArgMarshaler interface {
	Set(string) error
}

// Argument is used to define flags in our Command struct
type Argument struct {
	Name        string
	Description string
	Required    bool
	Value       interface{}
}
