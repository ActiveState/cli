package output

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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

func (f *Plain) Close() error {
	return nil
}

func (f *Plain) write(writer io.Writer, value interface{}) {
	v, err := sprint(value)
	if err != nil {
		logging.Errorf("Could not sprint value: %v, error: %v", value, err)
		writeColorized(fmt.Sprintf("[RED]%s[/RESET]", locale.Tr("err_sprint", err.Error())), f.cfg.ErrWriter)
		return
	}
	writeColorized(v, f.cfg.OutWriter)
}

func (f *Plain) writeNow(writer io.Writer, value string) {
	if f.cfg.Colored {
		writeColorized(value, writer)
	} else {
		writer.Write([]byte(value))
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

		key := locale.T(structMeta.serializedFields[i])
		result = append(result, fmt.Sprintf("%s: %s", key, stringValue))
	}
	return strings.Join(result, "\n"), nil
}

func sprintSlice(value interface{}) (string, error) {
	slice, err := parseSlice(value)
	if err != nil {
		return "", err
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
