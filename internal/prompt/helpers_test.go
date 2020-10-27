package prompt

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/output"
)

func TestInputRequired(t *testing.T) {
	assert.Error(t, inputRequired(""), "Throws error because value is empty")
	assert.NoError(t, inputRequired("foo"), "Doesn't throw an error cause value 'foo' is not empty")
	assert.NoError(t, inputRequired(0), "Doesn't throw an error cause value is '0' not empty")
	assert.NoError(t, inputRequired(false), "Doesn't throw an error cause value 'false' is not empty")
}

func TestFormatMessageByCols(t *testing.T) {
	assert.Equal(t, "aa\naa\naa", formatMessageByCols("aaaaaa", 3), "Adds linebreaks at col limit")
	assert.Equal(t, "a\naa\naa\na", formatMessageByCols("a\naaaaa", 3), "Adds linebreaks at col limit")
}

func TestPrompts(t *testing.T) {
	output.New(string(output.PlainFormatName), &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     true,
		Interactive: true,
	})
	p := New()

	fmt.Println("# SELECT")
	p.Select("Title", "This is the message", []string{"choice 1", "choice 2", "choice 3"}, "choice 1")

	fmt.Println("# CONFIRM")
	p.Confirm("Title", "This is the message", true)

	fmt.Println("# INPUT")
	p.Input("Title", "This is the message", "Default response")

	fmt.Println("# SECRET")
	p.InputSecret("Title", "This is the message")
}