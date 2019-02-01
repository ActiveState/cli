package surveyor

import (
	"errors"
	"reflect"

	"github.com/ActiveState/cli/internal/locale"
	survey "gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/core"
)

func init() {
	core.ErrorIcon = ""
	core.HelpIcon = ""
	core.QuestionIcon = ""
	core.ErrorTemplate = locale.Tt("survey_error_template")
}

// ValidateRequired does not allow an empty value
func ValidateRequired(val interface{}) error {
	// the reflect value of the result
	value := reflect.ValueOf(val)

	// if the value passed in is the zero value of the appropriate type
	if isZero(value) && value.Kind() != reflect.Bool {
		return errors.New(locale.T("err_value_required"))
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

// Confirm will prompt for a yes/no confirmation and return true if confirmed.
func Confirm(translationID string) (confirmed bool) {
	survey.AskOne(&survey.Confirm{
		Message: locale.T(translationID),
	}, &confirmed, nil)
	return confirmed
}
