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

func TestMultiSelectRender(t *testing.T) {

	prompt := MultiSelect{
		Message: "Pick your words:",
		Options: []string{"foo", "bar", "baz", "buz"},
		Default: []string{"bar", "buz"},
	}

	helpfulPrompt := prompt
	helpfulPrompt.Help = "This is helpful"

	tests := []struct {
		title    string
		prompt   MultiSelect
		data     MultiSelectTemplateData
		expected string
	}{
		{
			"Test MultiSelect question output",
			prompt,
			MultiSelectTemplateData{
				SelectedIndex: 2,
				PageEntries:   prompt.Options,
				Checked:       map[string]bool{"bar": true, "buz": true},
			},
			strings.Join(
				[]string{
					fmt.Sprintf("%s Pick your words:  [Use arrows to move, type to filter]", core.QuestionIcon),
					fmt.Sprintf("  %s  foo", core.UnmarkedOptionIcon),
					fmt.Sprintf("  %s  bar", core.MarkedOptionIcon),
					fmt.Sprintf("%s %s  baz", core.SelectFocusIcon, core.UnmarkedOptionIcon),
					fmt.Sprintf("  %s  buz\n", core.MarkedOptionIcon),
				},
				"\n",
			),
		},
		{
			"Test MultiSelect answer output",
			prompt,
			MultiSelectTemplateData{
				Answer:     "foo, buz",
				ShowAnswer: true,
			},
			fmt.Sprintf("%s Pick your words: foo, buz\n", core.QuestionIcon),
		},
		{
			"Test MultiSelect question output with help hidden",
			helpfulPrompt,
			MultiSelectTemplateData{
				SelectedIndex: 2,
				PageEntries:   prompt.Options,
				Checked:       map[string]bool{"bar": true, "buz": true},
			},
			strings.Join(
				[]string{
					fmt.Sprintf("%s Pick your words:  [Use arrows to move, type to filter, %s for more help]", core.QuestionIcon, string(core.HelpInputRune)),
					fmt.Sprintf("  %s  foo", core.UnmarkedOptionIcon),
					fmt.Sprintf("  %s  bar", core.MarkedOptionIcon),
					fmt.Sprintf("%s %s  baz", core.SelectFocusIcon, core.UnmarkedOptionIcon),
					fmt.Sprintf("  %s  buz\n", core.MarkedOptionIcon),
				},
				"\n",
			),
		},
		{
			"Test MultiSelect question output with help shown",
			helpfulPrompt,
			MultiSelectTemplateData{
				SelectedIndex: 2,
				PageEntries:   prompt.Options,
				Checked:       map[string]bool{"bar": true, "buz": true},
				ShowHelp:      true,
			},
			strings.Join(
				[]string{
					fmt.Sprintf("%s This is helpful", core.HelpIcon),
					fmt.Sprintf("%s Pick your words:  [Use arrows to move, type to filter]", core.QuestionIcon),
					fmt.Sprintf("  %s  foo", core.UnmarkedOptionIcon),
					fmt.Sprintf("  %s  bar", core.MarkedOptionIcon),
					fmt.Sprintf("%s %s  baz", core.SelectFocusIcon, core.UnmarkedOptionIcon),
					fmt.Sprintf("  %s  buz\n", core.MarkedOptionIcon),
				},
				"\n",
			),
		},
	}

	outputBuffer := bytes.NewBufferString("")
	terminal.Stdout = outputBuffer

	for _, test := range tests {
		outputBuffer.Reset()
		test.data.MultiSelect = test.prompt
		err := test.prompt.Render(
			MultiSelectQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)
		assert.Equal(t, test.expected, outputBuffer.String(), test.title)
	}
}
