package api_test

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/stretchr/testify/assert"
)

func TestErrorCode_WithoutPayload(t *testing.T) {
	assert.Equal(t, 100, api.ErrorCode(&struct{ Code int }{
		Code: 100,
	}))
}

func TestErrorCode_WithoutPayload_NoCodeValue(t *testing.T) {
	assert.Equal(t, -1, api.ErrorCode(&struct{ OtherCode int }{
		OtherCode: 100,
	}))
}

func TestErrorCode_WithPayload(t *testing.T) {
	providedCode := 200
	codeValue := struct{ Code *int }{Code: &providedCode}
	payload := struct{ Payload struct{ Code *int } }{
		Payload: codeValue,
	}

	assert.Equal(t, 200, api.ErrorCode(&payload))
}

func TestErrorCode_WithPayload_CodeNotPointer(t *testing.T) {
	providedCode := 300
	codeValue := struct{ Code int }{Code: providedCode}
	payload := struct{ Payload struct{ Code int } }{
		Payload: codeValue,
	}

	assert.Equal(t, 300, api.ErrorCode(&payload))
}

func TestErrorCode_WithPayload_NoCodeField(t *testing.T) {
	providedCode := 400
	codeValue := struct{ OtherCode int }{OtherCode: providedCode}
	payload := struct{ Payload struct{ OtherCode int } }{
		Payload: codeValue,
	}

	assert.Equal(t, -1, api.ErrorCode(&payload))
}
