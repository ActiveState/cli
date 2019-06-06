package survey

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ActiveState/survey/core"
	"github.com/ActiveState/survey/terminal"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestInputRender(t *testing.T) {

	tests := []struct {
		title    string
		prompt   Input
		data     InputTemplateData
		expected string
	}{
		{
			"Test Input question output without default",
			Input{Message: "What is your favorite month:"},
			InputTemplateData{},
			fmt.Sprintf("%s What is your favorite month: ", core.QuestionIcon),
		},
		{
			"Test Input question output with default",
			Input{Message: "What is your favorite month:", Default: "April"},
			InputTemplateData{},
			fmt.Sprintf("%s What is your favorite month: (April) ", core.QuestionIcon),
		},
		{
			"Test Input answer output",
			Input{Message: "What is your favorite month:"},
			InputTemplateData{Answer: "October", ShowAnswer: true},
			fmt.Sprintf("%s What is your favorite month: October\n", core.QuestionIcon),
		},
		{
			"Test Input question output without default but with help hidden",
			Input{Message: "What is your favorite month:", Help: "This is helpful"},
			InputTemplateData{},
			fmt.Sprintf("%s What is your favorite month: [%s for help] ", core.QuestionIcon, string(core.HelpInputRune)),
		},
		{
			"Test Input question output with default and with help hidden",
			Input{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			InputTemplateData{},
			fmt.Sprintf("%s What is your favorite month: [%s for help] (April) ", core.QuestionIcon, string(core.HelpInputRune)),
		},
		{
			"Test Input question output without default but with help shown",
			Input{Message: "What is your favorite month:", Help: "This is helpful"},
			InputTemplateData{ShowHelp: true},
			fmt.Sprintf("%s This is helpful\n%s What is your favorite month: ", core.HelpIcon, core.QuestionIcon),
		},
		{
			"Test Input question output with default and with help shown",
			Input{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			InputTemplateData{ShowHelp: true},
			fmt.Sprintf("%s This is helpful\n%s What is your favorite month: (April) ", core.HelpIcon, core.QuestionIcon),
		},
	}

	outputBuffer := bytes.NewBufferString("")
	terminal.Stdout = outputBuffer

	for _, test := range tests {
		outputBuffer.Reset()
		test.data.Input = test.prompt
		err := test.prompt.Render(
			InputQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)
		assert.Equal(t, test.expected, outputBuffer.String(), test.title)
	}
}
