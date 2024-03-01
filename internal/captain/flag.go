package captain

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/spf13/pflag"
)

type FlagMarshaler pflag.Value

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
			return errs.New("flag value must not be nil (%v)", flag)
		case *[]string:
			flagSetter().StringSliceVarP(
				v, flag.Name, flag.Shorthand, *v, flag.Description,
			)
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
			return errs.New(
				fmt.Sprintf("Unknown type for flag %s: %s (%v)", flag.Name, reflect.TypeOf(v).Name(), v),
			)
		}

		if flag.Hidden {
			if err := flagSetter().MarkHidden(flag.Name); err != nil {
				return errs.Wrap(err, "markFlagHidden %s failed", flag.Name)
			}
		}
	}

	return nil
}

type NullString struct {
	s     string
	isSet bool
}

func (s *NullString) String() string {
	if s.isSet {
		return s.s
	}
	return ""
}

func (s *NullString) Set(v string) error {
	s.s = v
	s.isSet = true
	return nil
}

func (s *NullString) Type() string {
	return "null string"
}

func (s *NullString) IsSet() bool {
	return s.isSet
}

func (s *NullString) AsPtrTo() *string {
	if !s.isSet {
		return nil
	}
	v := s.s
	return &v
}

type NullInt struct {
	n     int
	isSet bool
}

func (n *NullInt) String() string {
	if n.isSet {
		return strconv.FormatInt(int64(n.n), 10)
	}
	return ""
}

func (n *NullInt) Set(v string) error {
	x, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("null int set: %w", err)
	}

	n.n = x
	n.isSet = true

	return nil
}

func (n *NullInt) Type() string {
	return "null int"
}

func (n *NullInt) IsSet() bool {
	return n.isSet
}

func (n *NullInt) AsPtrTo() *int {
	if !n.isSet {
		return nil
	}
	v := n.n
	return &v
}

type NullBool struct {
	b     bool
	isSet bool
}

func (b *NullBool) String() string {
	if b.isSet {
		return fmt.Sprintf("%t", b.b)
	}
	return ""
}

func (b *NullBool) Set(v string) error {
	b.b = strings.ToLower(v) == "true"
	if !b.b && strings.ToLower(v) != "false" {
		return fmt.Errorf("null bool set: %q is not a valid value", v)
	}
	b.isSet = true
	return nil
}

func (b *NullBool) Type() string {
	return "null bool"
}

func (b *NullBool) IsSet() bool {
	return b.isSet
}

func (b *NullBool) AsPtrTo() *bool {
	if !b.isSet {
		return nil
	}
	v := b.b
	return &v
}

// LocalFlagUsages is a substitute for c.cobra.LocalFlags().FlagUsages() to replace Cobra's
// intuited flag value placeholders with "<value>".
func (c *Command) LocalFlagUsages() string {
	return flagUsagesWrapped(c.cobra.LocalFlags(), 0)
}

// InheritedFlagUsages is a substitute for c.cobra.InheritedFlags().FlagUsages() to replace Cobra's
// intuited flag value placeholders with "<value>".
func (c *Command) InheritedFlagUsages() string {
	return flagUsagesWrapped(c.cobra.InheritedFlags(), 0)
}

// flagUsagesWrapped is a copy of pflag.FlagSet.FlagUsagesWrapped(), but ignores Cobra's intuited
// flag value placeholders and uses "<value>" instead.
func flagUsagesWrapped(f *pflag.FlagSet, cols int) string {
	buf := new(bytes.Buffer)

	lines := make([]string, 0)

	maxlen := 0
	f.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}

		line := ""
		if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
			line = fmt.Sprintf("  -%s, --%s", flag.Shorthand, flag.Name)
		} else {
			line = fmt.Sprintf("      --%s", flag.Name)
		}

		// varname, usage := UnquoteUsage(flag)
		varname := ""
		if flag.Value.Type() != "bool" {
			varname = "<value>"
		}
		usage := flag.Usage
		if varname != "" {
			line += " " + varname
		}
		if flag.NoOptDefVal != "" {
			switch flag.Value.Type() {
			case "string":
				line += fmt.Sprintf("[=\"%s\"]", flag.NoOptDefVal)
			case "bool":
				if flag.NoOptDefVal != "true" {
					line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
				}
			case "count":
				if flag.NoOptDefVal != "+1" {
					line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
				}
			default:
				line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
			}
		}

		// This special character will be replaced with spacing once the
		// correct alignment is calculated
		line += "\x00"
		if len(line) > maxlen {
			maxlen = len(line)
		}

		line += usage
		//if !flag.defaultIsZeroValue() {
		//	if flag.Value.Type() == "string" {
		//		line += fmt.Sprintf(" (default %q)", flag.DefValue)
		//	} else {
		//		line += fmt.Sprintf(" (default %s)", flag.DefValue)
		//	}
		//}
		if len(flag.Deprecated) != 0 {
			line += fmt.Sprintf(" (DEPRECATED: %s)", flag.Deprecated)
		}

		lines = append(lines, line)
	})

	for _, line := range lines {
		sidx := strings.Index(line, "\x00")
		spacing := strings.Repeat(" ", maxlen-sidx)
		// maxlen + 2 comes from + 1 for the \x00 and + 1 for the (deliberate) off-by-one in maxlen-sidx
		fmt.Fprintln(buf, line[:sidx], spacing, wrap(maxlen+2, cols, line[sidx+1:]))
	}

	return buf.String()
}

// Splits the string `s` on whitespace into an initial substring up to
// `i` runes in length and the remainder. Will go `slop` over `i` if
// that encompasses the entire string (which allows the caller to
// avoid short orphan words on the final line).
func wrapN(i, slop int, s string) (string, string) {
	if i+slop > len(s) {
		return s, ""
	}

	w := strings.LastIndexAny(s[:i], " \t\n")
	if w <= 0 {
		return s, ""
	}
	nlPos := strings.LastIndex(s[:i], "\n")
	if nlPos > 0 && nlPos < w {
		return s[:nlPos], s[nlPos+1:]
	}
	return s[:w], s[w+1:]
}

// Wraps the string `s` to a maximum width `w` with leading indent
// `i`. The first line is not indented (this is assumed to be done by
// caller). Pass `w` == 0 to do no wrapping
func wrap(i, w int, s string) string {
	if w == 0 {
		return strings.Replace(s, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	// space between indent i and end of line width w into which
	// we should wrap the text.
	wrap := w - i

	var r, l string

	// Not enough space for sensible wrapping. Wrap as a block on
	// the next line instead.
	if wrap < 24 {
		i = 16
		wrap = w - i
		r += "\n" + strings.Repeat(" ", i)
	}
	// If still not enough space then don't even try to wrap.
	if wrap < 24 {
		return strings.Replace(s, "\n", r, -1)
	}

	// Try to avoid short orphan words on the final line, by
	// allowing wrapN to go a bit over if that would fit in the
	// remainder of the line.
	slop := 5
	wrap = wrap - slop

	// Handle first line, which is indented by the caller (or the
	// special case above)
	l, s = wrapN(wrap, slop, s)
	r = r + strings.Replace(l, "\n", "\n"+strings.Repeat(" ", i), -1)

	// Now wrap the rest
	for s != "" {
		var t string

		t, s = wrapN(wrap, slop, s)
		r = r + "\n" + strings.Repeat(" ", i) + strings.Replace(t, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	return r

}
