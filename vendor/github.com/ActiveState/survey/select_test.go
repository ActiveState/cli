package survey

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ActiveState/survey/core"
	"github.com/ActiveState/survey/terminal"
	"github.com/stretchr/testify/assert"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestSelectRender(t *testing.T) {

	prompt := Select{
		Message: "Pick your word:",
		Options: []string{"foo", "bar", "baz", "buz"},
		Default: "baz",
	}

	helpfulPrompt := prompt
	helpfulPrompt.Help = "This is helpful"

	tests := []struct {
		title    string
		prompt   Select
		data     SelectTemplateData
		expected string
	}{
		{
			"Test Select question output",
			prompt,
			SelectTemplateData{SelectedIndex: 2, PageEntries: prompt.Options},
			strings.Join(
				[]string{
					fmt.Sprintf("%s Pick your word:  [Use arrows to move, type to filter]", core.QuestionIcon),
					"  foo",
					"  bar",
					fmt.Sprintf("%s baz", core.SelectFocusIcon),
					"  buz\n",
				},
				"\n",
			),
		},
		{
			"Test Select answer output",
			prompt,
			SelectTemplateData{Answer: "buz", ShowAnswer: true, PageEntries: prompt.Options},
			fmt.Sprintf("%s Pick your word: buz\n", core.QuestionIcon),
		},
		{
			"Test Select question output with help hidden",
			helpfulPrompt,
			SelectTemplateData{SelectedIndex: 2, PageEntries: prompt.Options},
			strings.Join(
				[]string{
					fmt.Sprintf("%s Pick your word:  [Use arrows to move, type to filter, %s for more help]", core.QuestionIcon, string(core.HelpInputRune)),
					"  foo",
					"  bar",
					fmt.Sprintf("%s baz", core.SelectFocusIcon),
					"  buz\n",
				},
				"\n",
			),
		},
		{
			"Test Select question output with help shown",
			helpfulPrompt,
			SelectTemplateData{SelectedIndex: 2, ShowHelp: true, PageEntries: prompt.Options},
			strings.Join(
				[]string{
					fmt.Sprintf("%s This is helpful", core.HelpIcon),
					fmt.Sprintf("%s Pick your word:  [Use arrows to move, type to filter]", core.QuestionIcon),
					"  foo",
					"  bar",
					fmt.Sprintf("%s baz", core.SelectFocusIcon),
					"  buz\n",
				},
				"\n",
			),
		},
	}

	outputBuffer := bytes.NewBufferString("")
	terminal.Stdout = outputBuffer

	for _, test := range tests {
		outputBuffer.Reset()
		test.data.Select = test.prompt
		err := test.prompt.Render(
			SelectQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)
		assert.Equal(t, test.expected, outputBuffer.String(), test.title)
	}
}
