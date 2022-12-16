package output

// Simple is an outputer that works exactly as Plain without
// any notice level output
type Simple struct {
	Plain
}

// NewSimple constructs a new Simple struct
func NewSimple(config *Config) (Simple, error) {
	plain, err := NewPlain(config)
	if err != nil {
		return Simple{}, err
	}

	return Simple{plain}, nil
}

// Type tells callers what type of outputer we are
func (s *Simple) Type() Format {
	return SimpleFormatName
}

// Notice has no effect for this outputer
func (s *Simple) Notice(value interface{}) {}
