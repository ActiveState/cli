package prompt

import (
	"math"
	"reflect"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/core"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/osutils/termsize"

	"github.com/ActiveState/cli/internal/locale"
)

func init() {
	core.ErrorIcon = ""
	core.HelpIcon = ""
	core.QuestionIcon = ""
	core.SelectFocusIcon = ">"
	core.MarkedOptionIcon = "[x]"
	core.UnmarkedOptionIcon = "[ ]"
	core.ErrorTemplate = locale.Tt("survey_error_template")

	// Drop questionicon from templates as it causes indented text
	survey.SelectQuestionTemplate = `{{ .Message }}
{{- "\n"}}
{{- range $ix, $choice := .PageEntries}}
	{{- "\n"}}
	{{- if eq $ix $.SelectedIndex}}{{color "cyan+h"}}{{ SelectFocusIcon }} {{else}}  {{end}}
	{{- $choice}}
	{{- color "reset"}}
{{- end}}
`

	survey.InputQuestionTemplate = `{{- if ne .Message ""}}{{- .Message }}{{- "\n"}}{{- end}}
{{- color "cyan+h"}}{{- "> "}}{{- color "reset"}}`

	survey.ConfirmQuestionTemplate = `{{ .Message }}{{" "}}
{{- color "cyan+h"}}{{- if .Default}}(Y/n) {{- else}}(y/N) {{- end}}{{- color "reset"}}
{{color "cyan+h"}}{{- "> "}}{{- color "reset"}}`

	survey.PasswordQuestionTemplate = `{{- if ne .Message ""}}{{- .Message }}{{end}}
{{color "cyan+h"}}{{- "> "}}{{- color "reset"}}`
}

// inputRequired does not allow an empty value
func inputRequired(val interface{}) error {
	// the reflect value of the result
	value := reflect.ValueOf(val)

	// if the value passed in is the zero value of the appropriate type
	if isZero(value) && value.Kind() != reflect.Bool && value.Kind() != reflect.Int {
		return locale.NewError("err_value_required")
	}
	return nil
}

// isZero returns true if the passed value is the zero object
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	}

	// compare the types directly with more general coverage
	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}

func formatMessage(message string, colors bool) string {
	message = colorize.ColorizedOrStrip(message, colors)
	cols := termsize.GetTerminalColumns()
	return formatMessageByCols(message, cols)
}

func formatMessageByCols(message string, cols int) string {
	var newMessage string
	startIdx := 0
	cols = cols - 1 // reduce cols by 1 because the final col is the linebreak

	// Rebuild message and add linebreaks as needed
	for {
		if len(message[startIdx:]) == 0 {
			// EOF
			break
		}
		var idx int
		var endIdx = min(startIdx+cols, len(message))
		if idx = strings.Index(message[startIdx:endIdx], "\n"); idx == -1 {
			// If no linebreak was found move to the next column and add a linebreak
			idx = startIdx + min(cols, len(message[startIdx:]))
		} else {
			// Linebreak was found, move past it
			idx = startIdx + idx + 1 // Include the linebreak
		}

		newMessage = newMessage + message[startIdx:idx]
		startIdx = idx
		if len(message) > idx && newMessage[len(newMessage)-1:len(newMessage)] != "\n" {
			newMessage = newMessage + "\n"
		}
	}

	return newMessage
}

func min(v1 int, v2 int) int {
	return int(math.Min(float64(v1), float64(v2)))
}
