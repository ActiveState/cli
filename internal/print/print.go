package print

import (
	"fmt"
	"io"
	"os"

	"github.com/ActiveState/cli/internal/condition"
	ct "github.com/ActiveState/go-colortext"
)

// DisableColor is a flag that forces the print output to be monochrome
var DisableColor bool

// Printer holds our main printing logic
type Printer struct {
	output io.Writer
	plain  bool
}

// New creates a new Printer struct
func New(output io.Writer, plain bool) *Printer {
	return &Printer{output, plain}
}

func isPrinterPlain() bool {
	return condition.InTest() || DisableColor
}

// Stderr returns a new printer that uses stderr
func Stderr() *Printer {
	return New(os.Stderr, isPrinterPlain())
}

// Stdout returns a new printer that uses stdout, you can probably just use the methods exposed on this package instead
func Stdout() *Printer {
	return New(os.Stdout, isPrinterPlain())
}

// Line prints a formatted message and ends with a line break
func (p *Printer) Line(format string, a ...interface{}) {
	p.fprintfln(format, a...)
}

// Error prints the given string as an error message
func (p *Printer) Error(format string, a ...interface{}) {
	if p.plain == false {
		ct.Foreground(p.output, ct.Red, false)
		defer ct.Reset(p.output)
	}
	p.fprintfln(format, a...)
}

// Warning prints the given string as a warning message
func (p *Printer) Warning(format string, a ...interface{}) {
	if p.plain == false {
		ct.Foreground(p.output, ct.Yellow, false)
		defer ct.Reset(p.output)
	}
	p.fprintfln(format, a...)
}

// Info prints the given string as an info message
func (p *Printer) Info(format string, a ...interface{}) {
	if p.plain == false {
		ct.Foreground(p.output, ct.Blue, false)
		defer ct.Reset(p.output)
	}
	p.fprintfln(format, a...)
}

// Bold prints the given string as bolded message
func (p *Printer) Bold(format string, a ...interface{}) {
	if p.plain == false {
		ct.ChangeStyle(p.output, ct.Bold)
		defer ct.Reset(p.output)
	}
	p.fprintfln(format, a...)
}

// BoldInline prints the given string as bolded message in line
func (p *Printer) BoldInline(format string, a ...interface{}) {
	if p.plain == false {
		ct.ChangeStyle(p.output, ct.Bold)
		defer ct.Reset(p.output)
	}
	fmt.Fprintf(p.output, format, a...)
}

func (p *Printer) fprintfln(format string, a ...interface{}) {
	if len(a) == 0 {
		fmt.Fprintln(p.output, format)
	} else {
		fmt.Fprintf(p.output, format, a...)
		fmt.Fprintln(p.output)
	}
}

// Line prints a formatted message and ends with a line break
func Line(format string, a ...interface{}) {
	Stdout().Line(format, a...)
}

// Error prints the given string as an error message, error messages are always printed to stderr unless you use your own Printer
func Error(format string, a ...interface{}) {
	Stderr().Error(format, a...)
}

// Warning prints the given string as a warning message
func Warning(format string, a ...interface{}) {
	Stdout().Warning(format, a...)
}

// Info prints the given string as an info message
func Info(format string, a ...interface{}) {
	Stdout().Info(format, a...)
}

// Bold prints the given string as bolded message
func Bold(format string, a ...interface{}) {
	Stdout().Bold(format, a...)
}

// BoldInline prints the given string as bolded message in line
func BoldInline(format string, a ...interface{}) {
	Stdout().BoldInline(format, a...)
}
