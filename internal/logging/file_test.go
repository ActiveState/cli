package logging

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmit_Parallel(t *testing.T) {
	var messages []string
	for i := 0; i < 10; i++ {
		messages = append(messages, fmt.Sprintf("%d", i))
	}

	fh := newFileHandler()
	defer fh.Close()

	loggingFile, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	fh.file = loggingFile

	for i, message := range messages {
		t.Run(fmt.Sprintf("Parallel emit %d", i), func(t *testing.T) {
			t.Parallel()
			emitErr := fh.Emit(&MessageContext{}, message)
			assert.NoError(t, emitErr)
		})
	}
}
