package prompt

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	analyMock "github.com/ActiveState/cli/internal/analytics/mock"
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
	p := New(true, analyMock.New())

	fmt.Println("# SELECT")
	selectDefault := "choice 1"
	p.Select("Title", "This is the message", []string{"choice 1", "choice 2", "choice 3"}, &selectDefault)

	fmt.Println("# CONFIRM")
	confirmDefault := true
	p.Confirm("Title", "This is the message", &confirmDefault)

	fmt.Println("# INPUT")
	inputDefault := "Default response"
	p.Input("Title", "This is the message", &inputDefault)

	fmt.Println("# SECRET")
	p.InputSecret("Title", "This is the message")
}
