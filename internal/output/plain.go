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

// Plain ..
type Plain struct {
	cfg *Config
}

// NewPlain ..
func NewPlain(config *Config) (Plain, *failures.Failure) {
	return Plain{config}, nil
}

func (f *Plain) Print(value interface{}) {
	f.write(f.cfg.OutWriter, value)
}

func (f *Plain) Error(value interface{}) {
	f.write(f.cfg.ErrWriter, fmt.Sprintf("[RED]%s[/RESET]", value))
}

func (f *Plain) write(writer io.Writer, value interface{}) {
	v, err := sprint(value)
	if err != nil {
		logging.Errorf("Could not sprint value: %v, error: %v", value, err)
		f.writeNow(fmt.Sprintf("[RED]%s[/RESET]", locale.Tr("err_sprint", err.Error())), f.cfg.ErrWriter)
		return
	}
	f.writeNow(v, f.cfg.OutWriter)
}

func (f *Plain) writeNow(value string, writer io.Writer) {
	_, err := writeColorized(value, writer, !f.cfg.Colored)
	if err != nil {
		logging.Errorf("Writing colored output failed: %v", err)
	}
}

func sprint(value interface{}) (string, error) {
	var result string
	var err error

	valueRfl := reflect.ValueOf(value)
	kind := valueRfl.Kind()
	_ = kind
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

func localizedField(input string) string {
	return locale.T("field_" + strings.ToLower(input))
}
