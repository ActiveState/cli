package output

import (
	"fmt"
	"io"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

// PlainOpts define available tokens for setting plain output options.
type PlainOpts string

const (
	// SeparateLineOpt requests table output to be printed on a separate line (without columns)
	SeparateLineOpt PlainOpts = "separateLine"
	// EmptyNil replaces nil values with the empty string
	EmptyNil PlainOpts = "emptyNil"
	// HidePlain hides the field value in table output
	HidePlain PlainOpts = "hidePlain"
	// ShiftColsPrefix starts the column after the set qty
	ShiftColsPrefix PlainOpts = "shiftCols="
)

const dash = "\u2500"

// Plain is our plain outputer, it uses reflect to marshal the data.
// Semantic highlighting tags are supported as [NOTICE]foo[/RESET]
// Table output is supported if you pass a slice of structs
// Struct keys are localized by sending them to the locale library as field_key (lowercase)
type Plain struct {
	cfg *Config
}

// NewPlain constructs a new Plain struct
func NewPlain(config *Config) (Plain, *failures.Failure) {
	return Plain{config}, nil
}

// Type tells callers what type of outputer we are
func (f *Plain) Type() Format {
	return PlainFormatName
}

// Print will marshal and print the given value to the output writer
func (f *Plain) Print(value interface{}) {
	f.write(f.cfg.OutWriter, value)
	f.write(f.cfg.OutWriter, "\n")
}

// Error will marshal and print the given value to the error writer, it wraps it in the error format but otherwise the
// only thing that identifies it as an error is the channel it writes it to
func (f *Plain) Error(value interface{}) {
	f.write(f.cfg.ErrWriter, fmt.Sprintf("[ERROR]%s[/RESET]\n", value))
}

// Notice will marshal and print the given value to the error writer, it wraps it in the notice format but otherwise the
// only thing that identifies it as an error is the channel it writes it to
func (f *Plain) Notice(value interface{}) {
	f.write(f.cfg.ErrWriter, fmt.Sprintf("%s\n", value))
}

// Config returns the Config struct for the active instance
func (f *Plain) Config() *Config {
	return f.cfg
}

func (f *Plain) write(writer io.Writer, value interface{}) {
	v, err := sprint(value)
	if err != nil {
		logging.Errorf("Could not sprint value: %v, error: %v, stack: %s", value, err, stacktrace.Get().String())
		writeNow(f.cfg.ErrWriter, f.cfg.Colored, fmt.Sprintf("[ERROR]%s[/RESET]", locale.Tr("err_sprint", err.Error())))
		return
	}
	writeNow(writer, f.cfg.Colored, v)
}
