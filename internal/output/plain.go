package output

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/bndr/gotabulate"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type PlainOpts string

const (
	SingleLineOpt PlainOpts = "singleLine"
	EmptyNil      PlainOpts = "emptyNil"
)

// Plain is our plain outputer, it uses reflect to marshal the data.
// Color tags are supported as [RED]foo[/RESET]
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

// Error will marshal and print the given value to the error writer, it wraps it in red colored text but otherwise the
// only thing that identifies it as an error is the channel it writes it to
func (f *Plain) Error(value interface{}) {
	f.write(f.cfg.ErrWriter, fmt.Sprintf("[RED]%s[/RESET]\n", value))
}

// Notice will marshal and print the given value to the error writer, it wraps it in red colored text but otherwise the
// only thing that identifies it as an error is the channel it writes it to
func (f *Plain) Notice(value interface{}) {
	f.write(f.cfg.ErrWriter, value)
	f.write(f.cfg.ErrWriter, "\n")
}

// Config returns the Config struct for the active instance
func (f *Plain) Config() *Config {
	return f.cfg
}

// write is a little helper that just takes care of marshalling the value and sending it to the requested writer
func (f *Plain) write(writer io.Writer, value interface{}) {
	v, err := sprint(value)
	if err != nil {
		logging.Errorf("Could not sprint value: %v, error: %v", value, err)
		f.writeNow(f.cfg.ErrWriter, fmt.Sprintf("[RED]%s[/RESET]", locale.Tr("err_sprint", err.Error())))
		return
	}
	f.writeNow(writer, v)
}

// writeNow is a little helper that just writes the given value to the requested writer (no marshalling)
func (f *Plain) writeNow(writer io.Writer, value string) {
	_, err := writeColorized(value, writer, !f.cfg.Colored)
	if err != nil {
		logging.Errorf("Writing colored output failed: %v", err)
	}
}

const nilText = "<nil>"

// sprint will marshal and return the given value as a string
func sprint(value interface{}) (string, error) {
	if value == nil {
		return nilText, nil
	}

	if err, ok := value.(error); ok {
		return err.Error(), nil
	}

	valueRfl := valueOf(value)
	switch valueRfl.Kind() {
	case reflect.Ptr:
		if valueRfl.IsNil() {
			return nilText, nil
		}
		return sprint(valueRfl.Elem().Interface())

	case reflect.Struct:
		return sprintStruct(value)

	case reflect.Slice:
		if valueRfl.IsNil() {
			return nilText, nil
		}
		return sprintSlice(value)

	case reflect.Map:
		if valueRfl.IsNil() {
			return nilText, nil
		}
		return sprintMap(value)

	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", value), nil

	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", valueRfl.Float()), nil

	case reflect.Bool:
		return fmt.Sprintf("%t", valueRfl.Bool()), nil

	case reflect.String:
		if v, ok := value.(strfmt.UUID); ok {
			return v.String(), nil
		}
		return value.(string), nil

	default:
		return "", fmt.Errorf("unknown type: %s", valueRfl.Type().String())
	}
}

// sprintStruct will marshal and return the given struct as a string
func sprintStruct(value interface{}) (string, error) {
	meta, err := parseStructMeta(value)
	if err != nil {
		return "", err
	}

	result := []string{}
	for _, field := range meta {
		stringValue, err := sprint(field.value)
		if err != nil {
			return "", err
		}
		if stringValue == nilText {
			continue
		}

		if isStruct(field.value) || isSlice(field.value) {
			stringValue = "\n" + stringValue
		}

		key := localizedField(field.l10n)
		result = append(result, fmt.Sprintf("%s: %s", key, stringValue))
	}

	return strings.Join(result, "\n"), nil
}

// sprintSlice will marshal and return the given slice as a string
func sprintSlice(value interface{}) (string, error) {
	slice, err := parseSlice(value)
	if err != nil {
		return "", err
	}

	if len(slice) > 0 && isStruct(slice[0]) {
		return sprintTable(slice)
	}

	result := []string{}
	for _, v := range slice {
		stringValue, err := sprint(v)
		if err != nil {
			return "", err
		}

		// prepend if stringValue does not represent a slice
		if !isSlice(v) {
			stringValue = " - " + stringValue
		}

		result = append(result, stringValue)
	}

	return strings.Join(result, "\n"), nil
}

// sprintMap will marshal and return the given map as a string
func sprintMap(value interface{}) (string, error) {
	mp, err := parseMap(value)
	if err != nil {
		return "", err
	}

	result := []string{}
	for k, v := range mp {
		stringValue, err := sprint(v)
		if err != nil {
			return "", err
		}

		if isSlice(v) {
			stringValue = "\n" + stringValue
		}

		result = append(result, fmt.Sprintf(" %s: %s ", k, stringValue))
	}

	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })

	return "\n" + strings.Join(result, "\n"), nil
}

// sprintTable will marshal and return the given slice of structs as a string, formatted as a table
func sprintTable(slice []interface{}) (string, error) {
	if len(slice) == 0 {
		return "", nil
	}

	termWidth, _, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		logging.Debug("Cannot get terminal size: %v", err)
		termWidth = 100
	}

	headers := []string{}
	rows := [][]interface{}{}
	for _, v := range slice {
		if !isStruct(v) {
			return "", errors.New("Tried to sprintTable with slice that doesn't contain all structs")
		}

		meta, err := parseStructMeta(v)
		if err != nil {
			return "", err
		}

		firstIteration := len(headers) == 0
		row := []interface{}{}
		for _, field := range meta {
			if firstIteration {
				headers = append(headers, localizedField(field.l10n))
				termWidth = termWidth - (len(headers) * 10) // Account for cell padding, cause gotabulate doesn't..
			}

			stringValue, err := sprint(field.value)
			if err != nil {
				return "", err
			}

			if funk.Contains(field.opts, string(SingleLineOpt)) {
				stringValue = trimValue(stringValue, termWidth)
			}

			if funk.Contains(field.opts, string(EmptyNil)) && stringValue == nilText {
				stringValue = ""
			}

			row = append(row, stringValue)
		}

		rows = append(rows, row)
	}

	t := gotabulate.Create(rows)
	t.SetWrapDelimiter(' ')
	t.SetWrapStrings(true)
	t.SetMaxCellSize(termWidth)
	t.SetHeaders(headers)

	// Don't print whitespace lines
	t.SetHideLines([]string{"betweenLine", "top", "aboveTitle", "LineTop", "LineBottom", "bottomLine"})
	t.SetAlign("left")

	render := t.Render("simple")
	return strings.TrimSuffix(render, "\n"), nil
}

// localizedField is a little helper that will return the localized version of the given string
// locale values are in the form of `key,fallback` where fallback is optional
func localizedField(input string) string {
	in := strings.Split(input, ",") // First value is the locale key, second is the fallback value
	key := "field_" + strings.ToLower(input)
	out := locale.T("field_" + strings.ToLower(input))
	// If we can't find the locale for this field and this has a fallback value then use the fallback value
	if out == key && len(in) > 1 {
		out = in[1]
	}
	return out
}

func trimValue(value string, size int) string {
	value = strings.Replace(value, fileutils.LineEnd, " ", -1)
	if len(value) > size {
		value = value[0:size-5] + " [..]"
	}
	return value
}
