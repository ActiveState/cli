package output

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/table"
	"github.com/ActiveState/cli/internal/termutils"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

// PlainOpts define available tokens for setting plain output options.
type PlainOpts string

const (
	// SeparateLineOpt requests table output to be printed on a separate line (without columns)
	SeparateLineOpt PlainOpts = "separateLine"
	// VerticalTable requests a table be output vertically
	VerticalTable PlainOpts = "verticalTable"
	// EmptyNil replaces nil values with the empty string
	EmptyNil PlainOpts = "emptyNil"
	// HidePlain hides the field value in table output
	HidePlain PlainOpts = "hidePlain"
	// ShiftColsPrefix starts the column after the set qty
	ShiftColsPrefix PlainOpts = "shiftCols="
	// OmitEmpty omits empty values from output
	OmitEmpty PlainOpts = "omitEmpty"
	// HideDash hides the dash in table output
	HideDash PlainOpts = "hideDash"
	// OmitKey hides the current struct key from output, its value is still printed
	OmitKey PlainOpts = "omitKey"
)

// Plain is our plain outputer, it uses reflect to marshal the data.
// Semantic highlighting tags are supported as [NOTICE]foo[/RESET]
// Table output is supported if you pass a slice of structs
// Struct keys are localized by sending them to the locale library as field_key (lowercase)
type Plain struct {
	cfg *Config
}

// NewPlain constructs a new Plain struct
func NewPlain(config *Config) (Plain, error) {
	return Plain{config}, nil
}

// Type tells callers what type of outputer we are
func (f *Plain) Type() Format {
	return PlainFormatName
}

// Fprint allows printing to a specific writer, using all the conveniences of the output package
func (f *Plain) Fprint(writer io.Writer, v interface{}) {
	f.write(writer, v)
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

// write is a little helper that just takes care of marshalling the value and sending it to the requested writer
func (f *Plain) write(writer io.Writer, value interface{}) {
	v, err := sprint(value, nil)
	if err != nil {
		multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Could not sprint value: %v, error: %v, stack: %s", value, err, stacktrace.Get().String())
		f.writeNow(f.cfg.ErrWriter, fmt.Sprintf("[ERROR]%s[/RESET]", locale.Tr("err_sprint", err.Error())))
		return
	}
	f.writeNow(writer, v)
}

// writeNow is a little helper that just writes the given value to the requested writer (no marshalling)
func (f *Plain) writeNow(writer io.Writer, value string) {
	if f.Config().Interactive {
		value = colorize.Wrap(value, termutils.GetWidth(), true, "").String()
	}
	_, err := colorize.Colorize(value, writer, !f.cfg.Colored)
	if err != nil {
		logging.Warning("Writing colored output failed: %v", err)
	}
}

const nilText = "<nil>"

// sprint will marshal and return the given value as a string
func sprint(value interface{}, opts []string) (string, error) {
	if value == nil {
		return nilText, nil
	}

	switch t := value.(type) {
	case fmt.Stringer:
		return t.String(), nil
	case error:
		return t.Error(), nil
	case []byte: // Reflect doesn't handle []byte (easily)
		return string(t), nil
	}

	valueRfl := valueOf(value)
	switch valueRfl.Kind() {
	case reflect.Ptr:
		if valueRfl.IsNil() {
			return nilText, nil
		}
		return sprint(valueRfl.Elem().Interface(), opts)

	case reflect.Struct:
		return sprintStruct(value)

	case reflect.Slice:
		if valueRfl.IsNil() {
			return nilText, nil
		}
		return sprintSlice(value, opts)

	case reflect.Map:
		if valueRfl.IsNil() {
			return nilText, nil
		}
		return sprintMap(value, opts)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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
		if funk.Contains(field.opts, string(VerticalTable)) {
			slice, err := asSlice(field.value)
			if err != nil {
				return "", err
			}
			return sprintTable(true, false, slice)
		}

		stringValue, err := sprint(field.value, field.opts)
		if err != nil {
			return "", err
		}
		if stringValue == nilText {
			continue
		}

		if funk.Contains(field.opts, string(OmitKey)) {
			result = append(result, stringValue)
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
func sprintSlice(value interface{}, opts []string) (string, error) {
	slice, err := parseSlice(value)
	if err != nil {
		return "", err
	}

	if len(slice) > 0 && isStruct(slice[0]) {
		return sprintTable(false, funk.Contains(opts, string(HideDash)), slice)
	}

	result := []string{}
	for _, v := range slice {
		stringValue, err := sprint(v, opts)
		if err != nil {
			return "", err
		}

		// prepend if stringValue does not represent a slice
		if !isSlice(v) && !funk.Contains(opts, string(HideDash)) {
			stringValue = " - " + stringValue
		}

		result = append(result, stringValue)
	}

	return strings.Join(result, "\n"), nil
}

// sprintMap will marshal and return the given map as a string
func sprintMap(value interface{}, opts []string) (string, error) {
	mp, err := parseMap(value)
	if err != nil {
		return "", err
	}

	result := []string{}
	for k, v := range mp {
		stringValue, err := sprint(v, opts)
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
func sprintTable(vertical, hideDash bool, slice []interface{}) (string, error) {
	if len(slice) == 0 {
		return "", nil
	}

	if vertical {
		return sprintVerticalTable(slice)
	}

	headers := []string{}
	rows := [][]string{}
	for _, v := range slice {
		if !isStruct(v) {
			return "", errors.New("Tried to sprintTable with slice that doesn't contain all structs")
		}

		meta, err := parseStructMeta(v)
		if err != nil {
			return "", err
		}

		firstIteration := len(headers) == 0
		row := []string{}
		for _, field := range meta {
			if funk.Contains(field.opts, string(HidePlain)) {
				continue
			}

			stringValue, err := sprint(field.value, field.opts)
			if err != nil {
				return "", err
			}

			if funk.Contains(field.opts, string(OmitEmpty)) && (stringValue == "" || stringValue == nilText) {
				continue
			}

			if firstIteration && !funk.Contains(field.opts, string(SeparateLineOpt)) {
				headers = append(headers, localizedField(field.l10n))
			}

			if funk.Contains(field.opts, string(EmptyNil)) && stringValue == nilText {
				stringValue = ""
			}

			offset := shiftColsVal(field.opts)

			if funk.Contains(field.opts, string(SeparateLineOpt)) {
				rows = append(rows, row)
				if !funk.Contains(field.opts, string(EmptyNil)) || stringValue != "" {
					rows = append(rows, columns(offset, stringValue))
				}
				row = []string{}
				break
			}

			row = append(row, columns(offset, stringValue)...)
		}

		if len(row) > 0 {
			rows = append(rows, row)
		}
	}

	table := table.New(headers)
	if hideDash {
		table.HideDash = true
	}

	return table.AddRow(rows...).Render(), nil
}

type verticalRow struct {
	header  string
	content string
}

func sprintVerticalTable(slice []interface{}) (string, error) {
	if len(slice) == 0 {
		return "", nil
	}

	rows := [][]verticalRow{}
	for _, v := range slice {
		meta, err := parseStructMeta(v)
		if err != nil {
			return "", err
		}

		row := []verticalRow{}
		for _, field := range meta {
			if funk.Contains(field.opts, string(HidePlain)) {
				continue
			}

			stringValue, err := sprint(field.value, field.opts)
			if err != nil {
				return "", err
			}

			if funk.Contains(field.opts, string(OmitEmpty)) && (stringValue == "" || stringValue == nilText) {
				continue
			}

			if funk.Contains(field.opts, string(EmptyNil)) && stringValue == nilText {
				stringValue = ""
			}

			row = append(row, verticalRow{header: localizedField(field.l10n), content: stringValue})
		}

		if len(row) > 0 {
			rows = append(rows, row)
		}
	}

	t := table.New([]string{"", ""})
	t.AddRow(verticalRows(rows)...)
	t.HideHeaders = true
	t.Vertical = true
	return t.Render(), nil
}

func asSlice(val interface{}) ([]interface{}, error) {
	if !isSlice(val) {
		typeOf := reflect.TypeOf(val)
		collection := reflect.MakeSlice(reflect.SliceOf(typeOf), 0, 1)
		collection = reflect.Append(collection, valueOf(val))
		val = collection.Interface()
	}

	return parseSlice(val)
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

func shiftColsVal(opts []string) int {
	for _, opt := range opts {
		numChar := strings.TrimPrefix(opt, string(ShiftColsPrefix))
		if len(numChar) < len(opt) { // prefix exists and was trimmed
			n, err := strconv.Atoi(numChar)
			if err != nil {
				multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Cannot get shiftCols value: %v", err)
				break
			}

			return n
		}
	}

	return 0
}

func columns(offset int, value string) []string {
	if offset < 0 {
		offset = 0
		multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Negative shiftCols values are not handled; Using zero offset")
	}
	cols := make([]string, offset+1)
	cols[offset] = value
	return cols
}

func verticalRows(rows [][]verticalRow) [][]string {
	var vrows [][]string

	for i, hrow := range rows {
		for _, hcol := range hrow {
			vrow := []string{hcol.header, hcol.content}
			vrows = append(vrows, vrow)
		}

		if i < len(rows)-1 {
			vrows = append(vrows, []string{"", ""})
		}
	}

	return vrows
}
