package survey

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ActiveState/survey/core"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestPasswordRender(t *testing.T) {

	tests := []struct {
		title    string
		prompt   Password
		data     PasswordTemplateData
		expected string
	}{
		{
			"Test Password question output",
			Password{Message: "Tell me your secret:"},
			PasswordTemplateData{},
			fmt.Sprintf("%s Tell me your secret: ", core.QuestionIcon),
		},
		{
			"Test Password question output with help hidden",
			Password{Message: "Tell me your secret:", Help: "This is helpful"},
			PasswordTemplateData{},
			fmt.Sprintf("%s Tell me your secret: [%s for help] ", core.QuestionIcon, string(core.HelpInputRune)),
		},
		{
			"Test Password question output with help shown",
			Password{Message: "Tell me your secret:", Help: "This is helpful"},
			PasswordTemplateData{ShowHelp: true},
			fmt.Sprintf("%s This is helpful\n%s Tell me your secret: ", core.HelpIcon, core.QuestionIcon),
		},
	}

	for _, test := range tests {
		test.data.Password = test.prompt
		actual, err := core.RunTemplate(
			PasswordQuestionTemplate,
			&test.data,
		)
		assert.Nil(t, err, test.title)
		assert.Equal(t, test.expected, actual, test.title)
	}
}
