package output

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/bndr/gotabulate"
)

const PlainFormatName = "plain"

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

// Print will marshal and print the given value to the output writer
func (f *Plain) Print(value interface{}) {
	f.write(f.cfg.OutWriter, value)
}

// Error will marshal and print the given value to the error writer, it wraps it in red colored text but otherwise the
// only thing that identifies it as an error is the channel it writes it to
func (f *Plain) Error(value interface{}) {
	f.write(f.cfg.ErrWriter, fmt.Sprintf("[RED]%s[/RESET]", value))
}

// write is a little helper that just takes care of marshalling the value and sending it to the requested writer
func (f *Plain) write(writer io.Writer, value interface{}) {
	v, err := sprint(value)
	if err != nil {
		logging.Errorf("Could not sprint value: %v, error: %v", value, err)
		f.writeNow(fmt.Sprintf("[RED]%s[/RESET]", locale.Tr("err_sprint", err.Error())), f.cfg.ErrWriter)
		return
	}
	f.writeNow(v, f.cfg.OutWriter)
}

// writeNow is a little helper that just writes the given value to the requested writer (no marshalling)
func (f *Plain) writeNow(value string, writer io.Writer) {
	_, err := writeColorized(value, writer, !f.cfg.Colored)
	if err != nil {
		logging.Errorf("Writing colored output failed: %v", err)
	}
}

// sprint will marshal and return the given value as a string
func sprint(value interface{}) (string, error) {
	var result string
	var err error

	valueRfl := reflect.ValueOf(value)
	switch valueRfl.Kind() {
	case reflect.Ptr:
		return sprint(valueRfl.Elem().Interface())
	case reflect.Struct:
		var r string
		r, err = sprintStruct(value)
		result += r
	case reflect.Slice:
		var r string
		r, err = sprintSlice(value)
		result += r
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		result += fmt.Sprintf("%d", value)
	case reflect.Float32, reflect.Float64:
		result += fmt.Sprintf("%.2f", valueRfl.Float())
	case reflect.Bool:
		result += fmt.Sprintf("%t", valueRfl.Bool())
	case reflect.String:
		result += value.(string)
	default:
		err = fmt.Errorf("unknown type: %s", valueRfl.Type().String())
	}

	return result, err
}

// sprintStruct will marshal and return the given struct as a string
func sprintStruct(value interface{}) (string, error) {
	structMeta, err := parseStructMeta(value)
	if err != nil {
		return "", err
	}
	result := []string{}
	for i, value := range structMeta.values {
		stringValue, err := sprint(value)
		if err != nil {
			return "", err
		}

		key := localizedField(structMeta.localeFields[i])
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

	if len(slice) > 0 {
		if isStruct(slice[0]) {
			return sprintTable(slice)
		}
	}

	result := []string{}
	for _, v := range slice {
		stringValue, err := sprint(v)
		if err != nil {
			return "", err
		}

		result = append(result, stringValue)
	}

	return "\n - " + strings.Join(result, "\n - "), nil
}

// sprintTable will marshal and return the given slice of structs as a string, formatted as a table
func sprintTable(slice []interface{}) (string, error) {
	if len(slice) == 0 {
		return "", nil
	}

	headers := []string{}
	rows := [][]interface{}{}
	for _, v := range slice {
		if !isStruct(v) {
			return "", errors.New("Tried to sprintTable with slice that doesn't contain all structs")
		}

		structMeta, err := parseStructMeta(v)
		if err != nil {
			return "", err
		}

		setHeaders := len(headers) == 0
		row := []interface{}{}
		for i, value := range structMeta.values {
			stringValue, err := sprint(value)
			if err != nil {
				return "", err
			}

			row = append(row, stringValue)

			if setHeaders {
				headers = append(headers, localizedField(structMeta.localeFields[i]))
			}
		}

		rows = append(rows, row)
	}

	t := gotabulate.Create(rows)
	t.SetHeaders(headers)

	// Don't print whitespace lines
	t.SetHideLines([]string{"betweenLine", "top", "aboveTitle", "belowheader", "LineTop", "LineBottom", "bottomLine"})
	t.SetAlign("left")

	return t.Render("plain"), nil
}

// localizedField is a little helper that will return the localized version of the given string
func localizedField(input string) string {
	return locale.T("field_" + strings.ToLower(input))
}
